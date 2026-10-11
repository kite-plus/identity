//go:build e2e

package e2e

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/kite-plus/identity/internal/zitadel"
)

type received struct {
	header http.Header
	body   []byte
}

func collector(ch chan<- received) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ch <- received{r.Header.Clone(), b}
		w.WriteHeader(http.StatusOK)
	})
}

func wait(t *testing.T, ch <-chan received, d time.Duration, what string) received {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(d):
		t.Fatalf("no %s within %s", what, d)
		return received{}
	}
}

// fixture finds the readers org and the Explore project created by
// identity-init.
func (s *stack) fixture(t *testing.T, ctx context.Context) (orgID, projectID string) {
	t.Helper()
	orgs := call(t, ctx, s.admin, http.MethodPost, "/v2/organizations/_search", "", obj{
		"queries": []obj{{"nameQuery": obj{"name": "Kite Plus", "method": "TEXT_QUERY_METHOD_EQUALS"}}},
	})
	for _, o := range list(orgs, "result") {
		orgID = str(o, "id")
	}
	projects := call(t, ctx, s.admin, http.MethodPost, "/management/v1/projects/_search", orgID, obj{
		"queries": []obj{{"nameQuery": obj{"name": "Explore", "method": "TEXT_QUERY_METHOD_EQUALS"}}},
	})
	for _, p := range list(projects, "result") {
		projectID = str(p, "id")
	}
	if orgID == "" || projectID == "" {
		t.Fatal("readers org or project missing; run make dev-init")
	}
	return orgID, projectID
}

// TestBackChannelLogout registers a throwaway client whose back-channel
// logout URI points at a listener on this machine, signs in, ends the
// session the way the login UI's logout does, and validates the logout
// token.
func TestBackChannelLogout(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	orgID, projectID := s.fixture(t, ctx)

	ch := make(chan received, 4)
	base := listen(t, collector(ch))
	redirect := "http://localhost:1/callback"
	app := call(t, ctx, s.admin, http.MethodPost, "/management/v1/projects/"+projectID+"/apps/oidc", orgID, obj{
		"name":                 "e2e-" + randomString(4),
		"redirectUris":         []string{redirect},
		"responseTypes":        []string{"OIDC_RESPONSE_TYPE_CODE"},
		"grantTypes":           []string{"OIDC_GRANT_TYPE_AUTHORIZATION_CODE"},
		"appType":              "OIDC_APP_TYPE_WEB",
		"authMethodType":       "OIDC_AUTH_METHOD_TYPE_BASIC",
		"devMode":              true,
		"backChannelLogoutUri": base + "/auth/backchannel-logout",
		"loginVersion":         obj{"loginV2": obj{}},
	})
	appID := str(app, "appId")
	t.Cleanup(func() {
		_ = s.admin.Call(context.Background(), http.MethodDelete, "/management/v1/projects/"+projectID+"/apps/"+appID, orgID, nil, nil)
	})
	c := client{str(app, "clientId"), str(app, "clientSecret"), redirect}

	l := s.signIn(t, ctx, c, s.users["bob"])
	// What the login UI calls when the reader signs out at id.kite.plus.
	call(t, ctx, s.login, http.MethodDelete, "/v2/sessions/"+l.sessionID, "", obj{"sessionToken": l.sessionToken})

	r := wait(t, ch, 30*time.Second, "back-channel logout")
	if ct := r.header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		t.Errorf("content type %q", ct)
	}
	form, err := url.ParseQuery(string(r.body))
	if err != nil {
		t.Fatal(err)
	}
	raw := form.Get("logout_token")
	p := s.provider(t, ctx)
	tok, err := p.Verifier(&oidc.Config{ClientID: c.id}).Verify(ctx, raw)
	if err != nil {
		t.Fatalf("logout token: %v", err)
	}
	var claims struct {
		Sid    string                     `json:"sid"`
		Nonce  *string                    `json:"nonce"`
		Events map[string]json.RawMessage `json:"events"`
	}
	if err := tok.Claims(&claims); err != nil {
		t.Fatal(err)
	}
	if claims.Sid != l.claims["sid"] {
		t.Errorf("logout sid %q, ID token sid %v", claims.Sid, l.claims["sid"])
	}
	if tok.Subject != l.idToken.Subject {
		t.Errorf("logout sub %q, ID token sub %q", tok.Subject, l.idToken.Subject)
	}
	if _, ok := claims.Events["http://schemas.openid.net/event/backchannel-logout"]; !ok || claims.Nonce != nil {
		t.Errorf("not a logout token: %+v", claims)
	}
	t.Logf("logout token for sid %s, sub %s, user agent %q", claims.Sid, tok.Subject, r.header.Get("User-Agent"))
}

var userEvents = []string{"user.deactivated", "user.reactivated", "user.locked", "user.unlocked", "user.removed"}

// TestUserEvents adds a throwaway target to the user event executions,
// changes a throwaway user through every state, and checks the signed
// webhooks and the event query API with an app's sync token.
func TestUserEvents(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	orgID, _ := s.fixture(t, ctx)

	ch := make(chan received, 16)
	base := listen(t, collector(ch))
	target := call(t, ctx, s.admin, http.MethodPost, "/v2/actions/targets", "", obj{
		"name":        "e2e-" + randomString(4),
		"restWebhook": obj{"interruptOnError": false},
		"timeout":     "10s",
		"endpoint":    base + "/hooks/idp",
		"payloadType": "PAYLOAD_TYPE_JSON",
	})
	targetID, key := str(target, "id"), str(target, "signingKey")
	original := map[string][]string{}
	execs := call(t, ctx, s.admin, http.MethodPost, "/v2/actions/executions/search", "", obj{})
	for _, e := range list(execs, "executions") {
		original[str(e, "condition", "event", "event")] = toStrings(e["targets"])
	}
	for _, ev := range userEvents {
		call(t, ctx, s.admin, http.MethodPut, "/v2/actions/executions", "", obj{
			"condition": obj{"event": obj{"event": ev}},
			"targets":   append(slices.Clone(original[ev]), targetID),
		})
	}
	t.Cleanup(func() {
		bg := context.Background()
		for _, ev := range userEvents {
			_ = s.admin.Call(bg, http.MethodPut, "/v2/actions/executions", "", obj{
				"condition": obj{"event": obj{"event": ev}}, "targets": original[ev],
			}, nil)
		}
		_ = s.admin.Call(bg, http.MethodDelete, "/v2/actions/targets/"+targetID, "", nil, nil)
	})

	start := time.Now().Add(-2 * time.Second)
	name := "e2e-" + randomString(4)
	user := call(t, ctx, s.admin, http.MethodPost, "/v2/users/human", "", obj{
		"username":     name,
		"organization": obj{"orgId": orgID},
		"profile":      obj{"givenName": "E2E", "familyName": "User"},
		"email":        obj{"email": name + "@example.com", "isVerified": true},
	})
	userID := str(user, "userId")
	steps := []struct{ method, path, event string }{
		{http.MethodPost, "/v2/users/" + userID + "/deactivate", "user.deactivated"},
		{http.MethodPost, "/v2/users/" + userID + "/reactivate", "user.reactivated"},
		{http.MethodPost, "/v2/users/" + userID + "/lock", "user.locked"},
		{http.MethodPost, "/v2/users/" + userID + "/unlock", "user.unlocked"},
		{http.MethodDelete, "/v2/users/" + userID, "user.removed"},
	}
	for _, st := range steps {
		var in any = obj{}
		if st.method == http.MethodDelete {
			in = nil
		}
		call(t, ctx, s.admin, st.method, st.path, "", in)
		r := wait(t, ch, 30*time.Second, st.event+" webhook")
		if err := verifySignature(r.body, r.header.Get("ZITADEL-Signature"), key, 5*time.Minute); err != nil {
			t.Errorf("%s: %v", st.event, err)
		}
		var ev struct {
			AggregateID   string          `json:"aggregateID"`
			AggregateType string          `json:"aggregateType"`
			EventType     string          `json:"event_type"`
			ResourceOwner string          `json:"resourceOwner"`
			CreatedAt     string          `json:"created_at"`
			Payload       json.RawMessage `json:"event_payload"`
		}
		if err := json.Unmarshal(r.body, &ev); err != nil {
			t.Fatal(err)
		}
		if ev.EventType != st.event || ev.AggregateID != userID || ev.AggregateType != "user" {
			t.Errorf("webhook for %s: got %s", st.event, r.body)
		}
		t.Logf("%s webhook: signature %q body %s", st.event, r.header.Get("ZITADEL-Signature"), r.body)
	}

	// Reconciliation: every app's sync token can list the same events.
	for key, app := range s.apps {
		sync := zitadel.New(s.issuer, strings.TrimPrefix(s.issuer, "http://"), app.apiToken)
		got := map[string]bool{}
		deadline := time.Now().Add(20 * time.Second)
		for len(got) < len(userEvents) && time.Now().Before(deadline) {
			res := call(t, ctx, sync, http.MethodPost, "/admin/v1/events/_search", "", obj{
				"asc":            true,
				"limit":          1000,
				"from":           start.UTC().Format(time.RFC3339Nano),
				"aggregateTypes": []string{"user"},
				"eventTypes":     userEvents,
			})
			for _, e := range list(res, "events") {
				if str(e, "aggregate", "id") == userID {
					got[str(e, "type", "type")] = true
				}
			}
			if len(got) < len(userEvents) {
				time.Sleep(time.Second)
			}
		}
		for _, ev := range userEvents {
			if !got[ev] {
				t.Errorf("%s: event query misses %s (got %v)", key, ev, got)
			}
		}

		// Least privilege: the sync token must not be able to write.
		err := sync.Call(ctx, http.MethodPost, "/v2/users/human", "", obj{
			"username": "e2e-denied-" + randomString(3), "organization": obj{"orgId": orgID},
			"profile": obj{"givenName": "No", "familyName": "Way"}, "email": obj{"email": "denied@example.com"},
		}, nil)
		if e, ok := errors.AsType[*zitadel.APIError](err); !ok || e.Status != http.StatusForbidden {
			t.Errorf("%s: sync token could create a user or failed oddly: %v", key, err)
		}
	}
}

func toStrings(v any) []string {
	l, _ := v.([]any)
	out := make([]string, 0, len(l))
	for _, x := range l {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// verifySignature checks a ZITADEL-Signature header of the form
// "t=<unix seconds>,v1=<hex HMAC-SHA256 of "<t>.<body>">"; several v1 values
// may appear while a key is rotated.
func verifySignature(body []byte, header, key string, tolerance time.Duration) error {
	var ts string
	var sigs []string
	for part := range strings.SplitSeq(header, ",") {
		k, v, _ := strings.Cut(part, "=")
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("signature header without timestamp: " + header)
	}
	if d := time.Since(time.Unix(sec, 0)); d > tolerance || d < -tolerance {
		return errors.New("signature timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	want := mac.Sum(nil)
	for _, s := range sigs {
		if got, err := hex.DecodeString(s); err == nil && hmac.Equal(got, want) {
			return nil
		}
	}
	return errors.New("no matching v1 signature")
}
