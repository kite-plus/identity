//go:build e2e

// Package e2e checks the running identity-dev stack end to end, without a
// browser: OIDC code flow with PKCE, ID token claims, back-channel logout,
// signed user event webhooks and the event query API. Run `make dev-test`.
package e2e

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/identity/internal/zitadel"
)

const outDir = "../../dev/out"

type appEnv struct {
	key          string
	issuer       string
	clientID     string
	clientSecret string
	webhookKey   string
	apiToken     string
	redirectURI  string
}

type testUser struct {
	username, email, password, sub string
}

type stack struct {
	issuer string
	apps   map[string]appEnv
	users  map[string]testUser
	admin  *zitadel.Client
	login  *zitadel.Client
}

var redirects = map[string]string{
	"explore":  "http://localhost:8080/auth/callback",
	"comments": "http://localhost:8081/auth/callback",
}

func loadStack(t *testing.T) *stack {
	t.Helper()
	s := &stack{apps: map[string]appEnv{}, users: map[string]testUser{}}
	for key, redirect := range redirects {
		vars := readEnvFile(t, filepath.Join(outDir, key+".env"))
		p := strings.ToUpper(key) + "_"
		a := appEnv{
			key:          key,
			issuer:       vars[p+"OIDC_ISSUER"],
			clientID:     vars[p+"OIDC_CLIENT_ID"],
			clientSecret: vars[p+"OIDC_CLIENT_SECRET"],
			webhookKey:   vars[p+"IDP_WEBHOOK_SECRET"],
			apiToken:     vars[p+"IDP_API_TOKEN"],
			redirectURI:  redirect,
		}
		for name, v := range map[string]string{"issuer": a.issuer, "client id": a.clientID, "client secret": a.clientSecret, "webhook key": a.webhookKey, "api token": a.apiToken} {
			if v == "" {
				t.Fatalf("%s.env: %s is empty; run make dev-init", key, name)
			}
		}
		s.apps[key] = a
		s.issuer = a.issuer
	}

	f, err := os.Open(filepath.Join(outDir, "users.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			t.Fatalf("users.txt: bad line %q", line)
		}
		s.users[fields[0]] = testUser{fields[0], fields[1], fields[2], fields[3]}
	}

	host := strings.TrimPrefix(s.issuer, "http://")
	s.admin = zitadel.New(s.issuer, host, readFile(t, filepath.Join(outDir, "bootstrap", "admin.pat")))
	s.login = zitadel.New(s.issuer, host, readFile(t, filepath.Join(outDir, "bootstrap", "login-client.pat")))
	return s
}

func readEnvFile(t *testing.T, path string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for line := range strings.Lines(readFile(t, path)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok {
			out[k] = v
		}
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (is the stack initialized? run make dev-up dev-init)", err)
	}
	return strings.TrimSpace(string(b))
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type obj = map[string]any

func call(t *testing.T, ctx context.Context, c *zitadel.Client, method, path, org string, in any) obj {
	t.Helper()
	var out obj
	if err := c.Call(ctx, method, path, org, in, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func str(m obj, path ...string) string {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(obj)
		if !ok {
			return ""
		}
		cur = mm[p]
	}
	s, _ := cur.(string)
	return s
}

func list(m obj, key string) []obj {
	l, _ := m[key].([]any)
	out := make([]obj, 0, len(l))
	for _, x := range l {
		if o, ok := x.(obj); ok {
			out = append(out, o)
		}
	}
	return out
}
