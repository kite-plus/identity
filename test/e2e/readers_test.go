//go:build e2e

package e2e

import (
	"io"
	"net/http"
	"testing"

	"golang.org/x/oauth2"

	"github.com/kite-plus/identity/internal/apply"
	"github.com/kite-plus/identity/internal/config"
	"github.com/kite-plus/identity/internal/zitadel"
)

// zitadelAudience asks for an access token the ZITADEL API accepts.
const zitadelAudience = "urn:zitadel:iam:org:project:id:zitadel:aud"

// TestReaderSelfDeletion signs up a throwaway reader, lets identity-init's
// readers step grant the self-deletion role, and deletes the account with
// the reader's own access token, as an app's "delete my Kite Plus account"
// page would after a fresh sign-in.
func TestReaderSelfDeletion(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	orgID, _ := s.fixture(t, ctx)

	name := "e2e-del-" + randomString(4)
	password := "pw-" + randomString(8)
	user := call(t, ctx, s.admin, http.MethodPost, "/v2/users/human", "", obj{
		"username":     name,
		"organization": obj{"orgId": orgID},
		"profile":      obj{"givenName": "E2E", "familyName": "Delete"},
		"email":        obj{"email": name + "@example.com", "isVerified": true},
		"password":     obj{"password": password, "changeRequired": false},
	})
	userID := str(user, "userId")
	t.Cleanup(func() { _ = s.admin.Call(t.Context(), http.MethodDelete, "/v2/users/"+userID, "", nil, nil) })

	cfg, err := config.Load("../../config/dev.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apply.New(cfg, s.admin, "", io.Discard).RunReaders(ctx); err != nil {
		t.Fatal(err)
	}

	u := testUser{username: name, password: password}
	g := s.code(t, ctx, s.apps["explore"].client(), u, zitadelAudience)
	tok, err := g.conf.Exchange(ctx, g.code, oauth2.VerifierOption(g.verifier))
	if err != nil {
		t.Fatal(err)
	}
	own := *s.admin
	own.Token = tok.AccessToken
	if err := own.Call(ctx, http.MethodDelete, "/v2/users/"+userID, "", nil, nil); err != nil {
		t.Fatalf("reader could not delete their own account: %v", err)
	}
	if err := s.admin.Call(ctx, http.MethodGet, "/v2/users/"+userID, "", nil, nil); !zitadel.IsNotFound(err) {
		t.Errorf("user still readable after self-deletion: %v", err)
	}
}
