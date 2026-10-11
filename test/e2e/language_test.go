//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

// TestNewUserLanguage checks the reader language hook: a user created
// without a preferred language gets zh instead of ZITADEL's "und", and an
// explicit language is kept.
func TestNewUserLanguage(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	orgID, _ := s.fixture(t, ctx)
	for _, tc := range []struct{ given, want string }{{"", "zh"}, {"en", "en"}} {
		name := "e2e-lang-" + randomString(4)
		profile := obj{"givenName": "E2E", "familyName": "Lang"}
		if tc.given != "" {
			profile["preferredLanguage"] = tc.given
		}
		u := call(t, ctx, s.admin, http.MethodPost, "/v2/users/human", "", obj{
			"username": name, "organization": obj{"orgId": orgID}, "profile": profile,
			"email": obj{"email": name + "@example.com", "isVerified": true},
		})
		id := str(u, "userId")
		got := call(t, ctx, s.admin, http.MethodGet, "/v2/users/"+id, "", nil)
		if lang := str(got, "user", "human", "profile", "preferredLanguage"); lang != tc.want {
			t.Errorf("given %q: preferred language %q, want %q", tc.given, lang, tc.want)
		}
		_ = s.admin.Call(ctx, http.MethodDelete, "/v2/users/"+id, "", nil, nil)
	}
}
