//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"
)

// TestNoPasswordLockout makes sure failed passwords never lock an account:
// a permanent lock would let anyone shut a reader out.
func TestNoPasswordLockout(t *testing.T) {
	s := loadStack(t)
	ctx := t.Context()
	orgID, _ := s.fixture(t, ctx)
	name := "e2e-lock-" + randomString(4)
	u := call(t, ctx, s.admin, http.MethodPost, "/v2/users/human", "", obj{
		"username": name, "organization": obj{"orgId": orgID},
		"profile":  obj{"givenName": "E2E", "familyName": "Lock"},
		"email":    obj{"email": name + "@example.com", "isVerified": true},
		"password": obj{"password": "right-" + randomString(6), "changeRequired": false},
	})
	id := str(u, "userId")
	t.Cleanup(func() { _ = s.admin.Call(context.Background(), http.MethodDelete, "/v2/users/"+id, "", nil, nil) })
	for range 12 {
		err := s.login.Call(ctx, http.MethodPost, "/v2/sessions", "", obj{
			"checks": obj{"user": obj{"userId": id}, "password": obj{"password": "wrong"}},
		}, nil)
		if err == nil {
			t.Fatal("a wrong password was accepted")
		}
	}
	got := call(t, ctx, s.admin, http.MethodGet, "/v2/users/"+id, "", nil)
	if state := str(got, "user", "state"); state != "USER_STATE_ACTIVE" {
		t.Errorf("after 12 wrong passwords the user is %s", state)
	}
}
