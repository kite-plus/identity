//go:build e2e

package e2e

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"github.com/kite-plus/identity/internal/apply"
	"github.com/kite-plus/identity/internal/config"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/identity/internal/zitadel"
)

func TestAccountSessionAuthorization(t *testing.T) {
	stack := loadStack(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	user := stack.users["alice"]
	created := call(t, ctx, stack.login, http.MethodPost, "/v2/sessions", "", obj{
		"checks": obj{"user": obj{"loginName": user.username}, "password": obj{"password": user.password}},
	})
	sessionID := str(created, "sessionId")
	t.Cleanup(func() {
		_ = stack.login.Call(context.Background(), http.MethodDelete, "/v2/sessions/"+sessionID, "", nil, nil)
	})
	userClient := zitadel.New(stack.issuer, strings.TrimPrefix(stack.issuer, "http://"), str(created, "sessionToken"))
	for _, path := range []string{"/auth/v1/users/me", "/auth/v1/users/me/profile", "/v2/users/" + user.sub, "/auth/v1/users/me/email", "/auth/v1/policies/passwords/complexity", "/v2/users/" + user.sub + "/authentication_methods"} {
		var result obj
		if err := userClient.Call(ctx, http.MethodGet, path, "", nil, &result); err != nil {
			t.Errorf("authenticated account read %s: %v", path, err)
		}
	}
	var sessions obj
	if err := userClient.Call(ctx, http.MethodPost, "/v2/sessions/search", "", obj{"queries": []obj{{"userIdQuery": obj{"id": user.sub}}}}, &sessions); err != nil {
		t.Fatalf("list own sessions: %v", err)
	}
	for _, session := range list(sessions, "sessions") {
		if str(session, "factors", "user", "id") != user.sub {
			t.Fatal("session list leaked another user's session")
		}
	}
	var other obj
	if err := userClient.Call(ctx, http.MethodGet, "/v2/users/"+stack.users["bob"].sub, "", nil, &other); err == nil {
		t.Fatal("account session must not read another user's profile")
	}
	if err := userClient.Call(ctx, http.MethodDelete, "/v2/sessions/"+sessionID, "", nil, nil); err != nil {
		t.Fatalf("revoke own session: %v", err)
	}

}

func TestAccountSelfService(t *testing.T) {
	stack := loadStack(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	orgID, _ := stack.fixture(t, ctx)
	name := "e2e-account-" + randomString(4)
	password := "Account1!" + randomString(8)
	created := call(t, ctx, stack.admin, http.MethodPost, "/v2/users/human", "", obj{
		"username": name, "organization": obj{"orgId": orgID},
		"profile":  obj{"givenName": "E2E", "familyName": "Account", "displayName": "E2E Account"},
		"email":    obj{"email": name + "@example.com", "isVerified": true},
		"password": obj{"password": password, "changeRequired": false},
	})
	userID := str(created, "userId")
	t.Cleanup(func() {
		_ = stack.admin.Call(context.Background(), http.MethodDelete, "/v2/users/"+userID, "", nil, nil)
	})
	cfg, err := config.Load("../../config/dev.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apply.New(cfg, stack.admin, "", io.Discard).RunReaders(ctx); err != nil {
		t.Fatal(err)
	}
	session := call(t, ctx, stack.login, http.MethodPost, "/v2/sessions", "", obj{
		"checks": obj{"user": obj{"userId": userID}, "password": obj{"password": password}},
	})
	own := zitadel.New(stack.issuer, strings.TrimPrefix(stack.issuer, "http://"), str(session, "sessionToken"))
	call(t, ctx, own, http.MethodPut, "/auth/v1/users/me/profile", "", obj{
		"firstName": "E2E", "lastName": "Account", "displayName": "Updated Account", "preferredLanguage": "zh",
	})
	awaitAccountProjection(t, ctx, func() bool {
		profile := call(t, ctx, own, http.MethodGet, "/auth/v1/users/me/profile", "", nil)
		return str(profile, "profile", "displayName") == "Updated Account"
	})
	call(t, ctx, own, http.MethodPost, "/v2/users/"+userID+"/passkeys/_search", "", obj{})
	registration := call(t, ctx, own, http.MethodPost, "/v2/users/"+userID+"/passkeys", "", obj{"domain": "localhost"})
	if str(registration, "passkeyId") == "" {
		t.Fatal("passkey enrollment returned no id")
	}
	call(t, ctx, own, http.MethodDelete, "/v2/users/"+userID+"/passkeys/"+str(registration, "passkeyId"), "", nil)
	totp := call(t, ctx, own, http.MethodPost, "/v2/users/"+userID+"/totp", "", obj{})
	code := accountTOTP(t, str(totp, "secret"))
	call(t, ctx, own, http.MethodPost, "/v2/users/"+userID+"/totp/verify", "", obj{"code": code})
	updated := call(t, ctx, stack.login, http.MethodPatch, "/v2/sessions/"+str(session, "sessionId"), "", obj{
		"sessionToken": str(session, "sessionToken"), "checks": obj{"totp": obj{"code": code}},
	})
	own.Token = str(updated, "sessionToken")
	awaitAccountProjection(t, ctx, func() bool {
		methods := call(t, ctx, own, http.MethodGet, "/v2/users/"+userID+"/authentication_methods", "", nil)
		return strings.Contains(fmt.Sprint(methods["authMethodTypes"]), "AUTHENTICATION_METHOD_TYPE_TOTP")
	})
	call(t, ctx, own, http.MethodDelete, "/v2/users/"+userID+"/totp", "", nil)
	newPassword := "Changed1!" + randomString(8)
	call(t, ctx, own, http.MethodPut, "/auth/v1/users/me/password", "", obj{"oldPassword": password, "newPassword": newPassword})
	call(t, ctx, stack.login, http.MethodPost, "/v2/sessions", "", obj{
		"checks": obj{"user": obj{"userId": userID}, "password": obj{"password": newPassword}},
	})
	call(t, ctx, own, http.MethodPut, "/auth/v1/users/me/email", "", obj{"email": name + "-new@example.com"})
	email := call(t, ctx, own, http.MethodGet, "/auth/v1/users/me/email", "", nil)
	if str(email, "email", "email") != name+"-new@example.com" {
		t.Fatal("new email was not set")
	}
	if value, _ := email["email"].(map[string]any); value["isEmailVerified"] == true {
		t.Fatal("new email must remain unverified")
	}
	verification := call(t, ctx, stack.admin, http.MethodPost, "/v2/users/"+userID+"/email", "", obj{"email": name + "-verify@example.com", "returnCode": obj{}})
	call(t, ctx, own, http.MethodPost, "/auth/v1/users/me/email/_verify", "", obj{"code": str(verification, "verificationCode")})
	awaitAccountProjection(t, ctx, func() bool {
		if _, err := apply.New(cfg, stack.admin, "", io.Discard).RunReaders(ctx); err != nil {
			t.Fatal(err)
		}
		memberships := call(t, ctx, own, http.MethodPost, "/auth/v1/memberships/me/_search", "", obj{})
		return strings.Contains(fmt.Sprint(memberships), "ORG_USER_SELF_MANAGER")
	})
	call(t, ctx, own, http.MethodDelete, "/auth/v1/users/me", "", nil)
	if err := stack.admin.Call(ctx, http.MethodGet, "/v2/users/"+userID, "", nil, nil); !zitadel.IsNotFound(err) {
		t.Fatalf("deleted user is still readable: %v", err)
	}
}

func accountTOTP(t *testing.T, secret string) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatal(err)
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(time.Now().Unix()/30))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 15
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1000000)
}

// Account reads use async projections and may lag a successful write.
func awaitAccountProjection(t *testing.T, ctx context.Context, check func() bool) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		if check() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-deadline.C:
			t.Fatal("账号更新未同步到查询投影")
		case <-time.After(100 * time.Millisecond):
		}
	}
}
