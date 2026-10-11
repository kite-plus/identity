//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type client struct {
	id, secret, redirectURI string
}

func (a appEnv) client() client { return client{a.clientID, a.clientSecret, a.redirectURI} }

type login struct {
	rawIDToken   string
	idToken      *oidc.IDToken
	claims       map[string]any
	sessionID    string
	sessionToken string
}

// noRedirect behaves like a browser that stops at every redirect, so the
// test can read the Location header.
var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func (s *stack) provider(t *testing.T, ctx context.Context) *oidc.Provider {
	t.Helper()
	p, err := oidc.NewProvider(ctx, s.issuer)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// authorize starts an authorization request and returns the ID of the
// request ZITADEL hands to the login UI.
func (s *stack) authorize(t *testing.T, ctx context.Context, conf *oauth2.Config, state, nonce, verifier string, opts ...oauth2.AuthCodeOption) string {
	t.Helper()
	opts = append(opts, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	var loc *url.URL
	// A client created a moment ago may not be visible yet.
	for attempt := 0; ; attempt++ {
		resp, err := noRedirect.Get(conf.AuthCodeURL(state, opts...))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		loc, err = url.Parse(resp.Header.Get("Location"))
		if err == nil && resp.StatusCode == http.StatusFound && loc.Query().Has("authRequest") {
			break
		}
		if attempt == 10 {
			t.Fatalf("authorize: status %d location %q", resp.StatusCode, resp.Header.Get("Location"))
		}
		time.Sleep(500 * time.Millisecond)
	}
	id := loc.Query().Get("authRequest")
	if !strings.HasPrefix(id, "V2_") {
		t.Fatalf("authorize: no login v2 auth request in %s", loc)
	}
	return id
}

type grant struct {
	provider *oidc.Provider
	conf     *oauth2.Config
	code     string
	verifier string
	nonce    string
	session  obj
}

// code runs the authorization request for one user the way the login UI
// does: create a session with the password check and link it to the auth
// request. It returns the code from the callback.
func (s *stack) code(t *testing.T, ctx context.Context, c client, u testUser) grant {
	t.Helper()
	p := s.provider(t, ctx)
	conf := &oauth2.Config{ClientID: c.id, ClientSecret: c.secret, Endpoint: p.Endpoint(), RedirectURL: c.redirectURI, Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}
	state, nonce, verifier := randomString(8), randomString(8), oauth2.GenerateVerifier()
	reqID := s.authorize(t, ctx, conf, state, nonce, verifier)

	sess := call(t, ctx, s.login, http.MethodPost, "/v2/sessions", "", obj{
		"checks": obj{"user": obj{"loginName": u.username}, "password": obj{"password": u.password}},
	})
	cb := call(t, ctx, s.login, http.MethodPost, "/v2/oidc/auth_requests/"+reqID, "", obj{
		"session": obj{"sessionId": str(sess, "sessionId"), "sessionToken": str(sess, "sessionToken")},
	})
	cbURL, err := url.Parse(str(cb, "callbackUrl"))
	if err != nil {
		t.Fatal(err)
	}
	if got := cbURL.Scheme + "://" + cbURL.Host + cbURL.Path; got != c.redirectURI {
		t.Fatalf("callback went to %s, want %s", got, c.redirectURI)
	}
	if cbURL.Query().Get("state") != state {
		t.Fatalf("state mismatch in %s", cbURL)
	}
	return grant{p, conf, cbURL.Query().Get("code"), verifier, nonce, sess}
}

// signIn runs the whole code flow and verifies the ID token.
func (s *stack) signIn(t *testing.T, ctx context.Context, c client, u testUser) login {
	t.Helper()
	g := s.code(t, ctx, c, u)
	tok, err := g.conf.Exchange(ctx, g.code, oauth2.VerifierOption(g.verifier))
	if err != nil {
		t.Fatal(err)
	}
	return s.verify(t, ctx, g.provider, c, tok, g.nonce, g.session)
}

func (s *stack) verify(t *testing.T, ctx context.Context, p *oidc.Provider, c client, tok *oauth2.Token, nonce string, sess obj) login {
	t.Helper()
	raw, _ := tok.Extra("id_token").(string)
	idt, err := p.Verifier(&oidc.Config{ClientID: c.id}).Verify(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	if idt.Nonce != nonce {
		t.Fatalf("nonce %q, want %q", idt.Nonce, nonce)
	}
	var claims map[string]any
	if err := idt.Claims(&claims); err != nil {
		t.Fatal(err)
	}
	return login{raw, idt, claims, str(sess, "sessionId"), str(sess, "sessionToken")}
}

func TestDiscovery(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	var meta struct {
		Issuer                string   `json:"issuer"`
		BackchannelLogout     bool     `json:"backchannel_logout_supported"`
		BackchannelSession    bool     `json:"backchannel_logout_session_supported"`
		CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
		EndSessionEndpoint    string   `json:"end_session_endpoint"`
		SubjectTypesSupported []string `json:"subject_types_supported"`
	}
	if err := s.provider(t, ctx).Claims(&meta); err != nil {
		t.Fatal(err)
	}
	if meta.Issuer != s.issuer {
		t.Errorf("issuer %q, want %q", meta.Issuer, s.issuer)
	}
	if !meta.BackchannelLogout || !meta.BackchannelSession {
		t.Errorf("back-channel logout not advertised: %+v", meta)
	}
	if !strings.Contains(strings.Join(meta.CodeChallengeMethods, " "), "S256") {
		t.Errorf("S256 not supported: %v", meta.CodeChallengeMethods)
	}
	t.Logf("discovery: %+v", meta)
}

func TestCodeFlowAndClaims(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	alice := s.users["alice"]
	subs := map[string]string{}
	for key, app := range s.apps {
		l := s.signIn(t, ctx, app.client(), alice)
		for _, claim := range []string{"sub", "sid", "email", "email_verified", "name", "preferred_username", "locale"} {
			if _, ok := l.claims[claim]; !ok {
				t.Errorf("%s: ID token lacks %s: %v", key, claim, l.claims)
			}
		}
		if l.claims["email_verified"] != true {
			t.Errorf("%s: email_verified = %v", key, l.claims["email_verified"])
		}
		if l.claims["sid"] != l.sessionID {
			t.Errorf("%s: sid %v is not the session ID %s", key, l.claims["sid"], l.sessionID)
		}
		for other, oa := range s.apps {
			if other != key && slices.Contains(l.idToken.Audience, oa.clientID) {
				t.Errorf("%s: ID token audience includes the %s client: %v", key, other, l.idToken.Audience)
			}
		}
		if l.claims["azp"] != app.clientID {
			t.Errorf("%s: azp %v, want %s", key, l.claims["azp"], app.clientID)
		}
		subs[key] = l.idToken.Subject
		t.Logf("%s: ID token claims %v", key, l.claims)
	}
	if subs["explore"] != subs["comments"] || subs["explore"] != alice.sub {
		t.Errorf("sub differs across apps: %v (users.txt says %s)", subs, alice.sub)
	}
}

func TestPKCEEnforced(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	g := s.code(t, ctx, s.apps["comments"].client(), s.users["bob"])
	if _, err := g.conf.Exchange(ctx, g.code, oauth2.VerifierOption(oauth2.GenerateVerifier())); err == nil {
		t.Fatal("token endpoint accepted a wrong PKCE verifier")
	}
	g = s.code(t, ctx, s.apps["comments"].client(), s.users["bob"])
	if _, err := g.conf.Exchange(ctx, g.code); err == nil {
		t.Fatal("token endpoint accepted a missing PKCE verifier")
	}
}

// TestPromptNoneWithoutSession documents login v2 behaviour: OIDC requires
// a redirect with error=login_required, ZITADEL v4.19.4 shows the login
// page instead (zitadel/zitadel#12165).
func TestPromptNoneWithoutSession(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	app := s.apps["explore"]
	p := s.provider(t, ctx)
	conf := &oauth2.Config{ClientID: app.clientID, ClientSecret: app.clientSecret, Endpoint: p.Endpoint(), RedirectURL: app.redirectURI, Scopes: []string{oidc.ScopeOpenID}}
	reqID := s.authorize(t, ctx, conf, "st", "nc", oauth2.GenerateVerifier(), oauth2.SetAuthURLParam("prompt", "none"))
	resp, err := noRedirect.Get(s.issuer + "/ui/v2/login/login?authRequest=" + reqID)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	switch {
	case strings.HasPrefix(loc, app.redirectURI) && strings.Contains(loc, "error=login_required"):
		t.Logf("prompt=none returns login_required as specified: %s", loc)
	case strings.Contains(loc, "/loginname"):
		t.Logf("KNOWN GAP: prompt=none without a session shows the login page (%d %s)", resp.StatusCode, loc)
	default:
		t.Errorf("unexpected prompt=none answer: %d %s", resp.StatusCode, loc)
	}
}

// listen starts an HTTP server on 127.0.0.1 and returns the URL the
// ZITADEL container reaches it at.
func listen(t *testing.T, h http.Handler) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Log(err)
		}
	}()
	t.Cleanup(func() { srv.Close() })
	return fmt.Sprintf("http://host.docker.internal:%d", ln.Addr().(*net.TCPAddr).Port)
}
