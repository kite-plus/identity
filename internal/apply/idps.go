package apply

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"slices"
)

// providerOptions lets readers sign up with a provider without a password
// and link it to an existing account with the same verified email.
var providerOptions = obj{
	"isLinkingAllowed":  true,
	"isCreationAllowed": true,
	"isAutoCreation":    true,
	"isAutoUpdate":      true,
	"autoLinking":       "AUTO_LINKING_OPTION_EMAIL",
}

func (a *Applier) idps(ctx context.Context) error {
	var tpl obj
	if err := a.call(ctx, http.MethodPost, "/admin/v1/idps/templates/_search", "", obj{}, &tpl); err != nil {
		return err
	}
	byName := map[string]string{}
	for _, t := range list(tpl, "result") {
		byName[str(t, "name")] = str(t, "id")
	}
	var enabled []string

	if g := a.cfg.IdPs.GitHub; g.Enabled() {
		in := obj{"name": g.Name, "clientId": g.ClientID, "scopes": g.Scopes, "providerOptions": providerOptions}
		id, err := a.upsertIdP(ctx, "github", byName[g.Name], in, obj{"clientSecret": g.ClientSecret}, hash(g.ClientSecret))
		if err != nil {
			return err
		}
		enabled = append(enabled, id)
	} else {
		a.note("GitHub sign-in not configured (GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET), skipped")
	}

	if ap := a.cfg.IdPs.Apple; ap.Enabled() {
		key, err := os.ReadFile(a.cfg.Path(ap.PrivateKeyFile))
		if err != nil {
			return err
		}
		in := obj{"name": ap.Name, "clientId": ap.ClientID, "teamId": ap.TeamID, "keyId": ap.KeyID, "scopes": ap.Scopes, "providerOptions": providerOptions}
		id, err := a.upsertIdP(ctx, "apple", byName[ap.Name], in, obj{"privateKey": base64.StdEncoding.EncodeToString(key)}, hash(string(key)))
		if err != nil {
			return err
		}
		enabled = append(enabled, id)
	} else {
		a.note("Apple sign-in not configured (APPLE_CLIENT_ID, APPLE_TEAM_ID, APPLE_KEY_ID, APPLE_PRIVATE_KEY_FILE), skipped")
	}

	var linked obj
	if err := a.call(ctx, http.MethodPost, "/admin/v1/policies/login/idps/_search", "", obj{}, &linked); err != nil {
		return err
	}
	var have []string
	for _, l := range list(linked, "result") {
		have = append(have, str(l, "idpId"))
	}
	for _, id := range enabled {
		if !slices.Contains(have, id) {
			if err := a.call(ctx, http.MethodPost, "/admin/v1/policies/login/idps", "", obj{"idpId": id}, nil); err != nil {
				return err
			}
			a.changed("identity provider %s added to the login policy", id)
		}
	}
	return nil
}

// upsertIdP creates or updates one provider. Secret fields cannot be read
// back, so they are sent only on create or when their hash changed.
func (a *Applier) upsertIdP(ctx context.Context, kind, id string, fields, secret obj, secretHash string) (string, error) {
	stateKey := "idp:" + kind
	if id == "" {
		var out obj
		if err := a.call(ctx, http.MethodPost, "/admin/v1/idps/"+kind, "", merged(fields, secret), &out); err != nil {
			return "", err
		}
		a.state.Hashes[stateKey] = secretHash
		a.changed("identity provider %s created", kind)
		return str(out, "id"), nil
	}
	var cur obj
	if err := a.call(ctx, http.MethodGet, "/admin/v1/idps/"+id, "", nil, &cur); err != nil {
		return "", err
	}
	idp := sub(cur, "idp")
	cfg := sub(sub(idp, "config"), kind)
	have := merged(cfg, obj{"name": idp["name"], "providerOptions": sub(sub(idp, "config"), "options")})
	update := len(diff(fields, have)) > 0
	in := fields
	if a.state.Hashes[stateKey] != secretHash {
		in = merged(fields, secret)
		update = true
	}
	if update {
		if err := a.call(ctx, http.MethodPut, "/admin/v1/idps/"+kind+"/"+id, "", in, nil); err != nil {
			return "", err
		}
		a.state.Hashes[stateKey] = secretHash
		a.changed("identity provider %s updated", kind)
	}
	return id, nil
}
