package apply

import (
	"context"
	"net/http"

	"github.com/kite-plus/identity/internal/config"
)

func (a *Applier) projectAndApps(ctx context.Context) error {
	for _, app := range a.cfg.Apps {
		projectID, err := a.project(ctx, app.Project)
		if err != nil {
			return err
		}
		var apps obj
		if err := a.call(ctx, http.MethodPost, "/management/v1/projects/"+projectID+"/apps/_search", a.readersID, obj{}, &apps); err != nil {
			return err
		}
		var cur obj
		for _, ap := range list(apps, "result") {
			if str(ap, "name") == app.Name {
				cur = ap
			}
		}
		if err := a.app(ctx, projectID, app, cur); err != nil {
			return err
		}
	}
	return nil
}

func (a *Applier) project(ctx context.Context, name string) (string, error) {
	var out obj
	in := obj{"queries": []obj{{"nameQuery": obj{"name": name, "method": textEquals}}}}
	if err := a.call(ctx, http.MethodPost, "/management/v1/projects/_search", a.readersID, in, &out); err != nil {
		return "", err
	}
	for _, p := range list(out, "result") {
		if str(p, "name") == name {
			return str(p, "id"), nil
		}
	}
	var created obj
	// No role checks: any Kite Plus account may sign in to every app.
	req := obj{"name": name, "projectRoleAssertion": false, "projectRoleCheck": false, "hasProjectCheck": false}
	if err := a.call(ctx, http.MethodPost, "/management/v1/projects", a.readersID, req, &created); err != nil {
		return "", err
	}
	a.changed("project %q created", name)
	return str(created, "id"), nil
}

// oidcConfig is the desired client configuration: a confidential web
// client using the authorization code flow (PKCE is accepted on top),
// ID token carrying the profile and email claims, no refresh tokens.
func oidcConfig(app config.App) obj {
	return obj{
		"redirectUris":             app.RedirectURIs,
		"responseTypes":            []string{"OIDC_RESPONSE_TYPE_CODE"},
		"grantTypes":               []string{"OIDC_GRANT_TYPE_AUTHORIZATION_CODE"},
		"appType":                  "OIDC_APP_TYPE_WEB",
		"authMethodType":           "OIDC_AUTH_METHOD_TYPE_BASIC",
		"postLogoutRedirectUris":   app.PostLogoutRedirectURIs,
		"devMode":                  app.DevMode,
		"accessTokenType":          "OIDC_TOKEN_TYPE_BEARER",
		"accessTokenRoleAssertion": false,
		"idTokenRoleAssertion":     false,
		"idTokenUserinfoAssertion": true,
		"clockSkew":                "0s",
		"backChannelLogoutUri":     app.BackChannelLogoutURI,
		"loginVersion":             obj{"loginV2": obj{}},
	}
}

func (a *Applier) app(ctx context.Context, projectID string, app config.App, cur obj) error {
	want := oidcConfig(app)
	st := a.state.Apps[app.Key]
	if cur == nil {
		in := merged(want, obj{"name": app.Name, "version": "OIDC_VERSION_1_0"})
		var out obj
		if err := a.call(ctx, http.MethodPost, "/management/v1/projects/"+projectID+"/apps/oidc", a.readersID, in, &out); err != nil {
			return err
		}
		a.state.Apps[app.Key] = AppState{AppID: str(out, "appId"), ClientID: str(out, "clientId"), ClientSecret: str(out, "clientSecret")}
		a.changed("app %q created", app.Name)
		return nil
	}
	appID := str(cur, "id")
	// Zero enum values are omitted from responses.
	have := merged(obj{
		"appType":         "OIDC_APP_TYPE_WEB",
		"authMethodType":  "OIDC_AUTH_METHOD_TYPE_BASIC",
		"accessTokenType": "OIDC_TOKEN_TYPE_BEARER",
	}, sub(cur, "oidcConfig"))
	if d := diff(want, have); len(d) > 0 {
		if err := a.call(ctx, http.MethodPut, "/management/v1/projects/"+projectID+"/apps/"+appID+"/oidc_config", a.readersID, want, nil); err != nil {
			return err
		}
		a.changed("app %q %v", app.Name, d)
	}
	clientID := str(have, "clientId")
	if st.ClientID != clientID || st.ClientSecret == "" {
		var out obj
		if err := a.call(ctx, http.MethodPost, "/management/v1/projects/"+projectID+"/apps/"+appID+"/oidc_config/_generate_client_secret", a.readersID, obj{}, &out); err != nil {
			return err
		}
		a.state.Apps[app.Key] = AppState{AppID: appID, ClientID: clientID, ClientSecret: str(out, "clientSecret")}
		a.changed("app %q client secret regenerated (not in local state)", app.Name)
	}
	return nil
}
