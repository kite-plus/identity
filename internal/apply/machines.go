package apply

import (
	"context"
	"net/http"
	"slices"
	"time"
)

func machineName(appKey string) string { return appKey + "-sync" }

// machines creates one machine user per app with a personal access token
// for reconciliation: reading users and the event stream. ZITADEL has no
// built-in role narrower than IAM_OWNER_VIEWER that grants events.read.
func (a *Applier) machines(ctx context.Context) error {
	for _, app := range a.cfg.Apps {
		if err := a.machine(ctx, app.Key, app.Name); err != nil {
			return err
		}
	}
	return nil
}

func (a *Applier) machine(ctx context.Context, key, appName string) error {
	username := machineName(key)
	userID, err := a.findUser(ctx, username)
	if err != nil {
		return err
	}
	if userID == "" {
		in := obj{
			"userName":        username,
			"name":            appName + " sync",
			"description":     "Reads users and user events for " + appName + " reconciliation",
			"accessTokenType": "ACCESS_TOKEN_TYPE_BEARER",
		}
		var out obj
		if err := a.call(ctx, http.MethodPost, "/management/v1/users/machine", a.operatorsID, in, &out); err != nil {
			return err
		}
		userID = str(out, "userId")
		a.changed("machine user %s created", username)
	}

	role := a.cfg.SyncRole
	var members obj
	in := obj{"queries": []obj{{"userIdQuery": obj{"userId": userID}}}}
	if err := a.call(ctx, http.MethodPost, "/admin/v1/members/_search", "", in, &members); err != nil {
		return err
	}
	var roles []string
	found := false
	for _, m := range list(members, "result") {
		if str(m, "userId") == userID {
			found = true
			roles = strs(m["roles"])
		}
	}
	switch {
	case !found:
		if err := a.call(ctx, http.MethodPost, "/admin/v1/members", "", obj{"userId": userID, "roles": []string{role}}, nil); err != nil {
			return err
		}
		a.changed("machine user %s granted %s", username, role)
	case !slices.Equal(roles, []string{role}):
		if err := a.call(ctx, http.MethodPut, "/admin/v1/members/"+userID, "", obj{"roles": []string{role}}, nil); err != nil {
			return err
		}
		a.changed("machine user %s roles set to %s", username, role)
	}

	st := a.state.Machines[key]
	var pats obj
	if err := a.call(ctx, http.MethodPost, "/management/v1/users/"+userID+"/pats/_search", a.operatorsID, obj{}, &pats); err != nil {
		return err
	}
	valid := false
	for _, p := range list(pats, "result") {
		if str(p, "id") != st.PATID {
			continue
		}
		exp, err := time.Parse(time.RFC3339Nano, str(p, "expirationDate"))
		valid = err == nil && time.Until(exp) > 30*24*time.Hour
	}
	if st.UserID == userID && valid && st.Token != "" {
		return nil
	}
	lifetime := a.cfg.SyncTokenLifetime.Std()
	if lifetime == 0 {
		lifetime = 365 * 24 * time.Hour
	}
	exp := time.Now().Add(lifetime).UTC().Format(time.RFC3339)
	var out obj
	if err := a.call(ctx, http.MethodPost, "/management/v1/users/"+userID+"/pats", a.operatorsID, obj{"expirationDate": exp}, &out); err != nil {
		return err
	}
	a.state.Machines[key] = MachineState{UserID: userID, PATID: str(out, "tokenId"), Token: str(out, "token")}
	a.changed("machine user %s token issued, expires %s", username, exp)
	if st.PATID != "" && st.UserID == userID {
		// Revoke the token this one replaces.
		if err := a.call(ctx, http.MethodDelete, "/management/v1/users/"+userID+"/pats/"+st.PATID, a.operatorsID, nil, nil); err != nil && !isNotFound(err) {
			return err
		}
	}
	return nil
}

func (a *Applier) findUser(ctx context.Context, username string) (string, error) {
	var out obj
	in := obj{"queries": []obj{{"userNameQuery": obj{"userName": username, "method": textEquals}}}}
	if err := a.call(ctx, http.MethodPost, "/v2/users", "", in, &out); err != nil {
		return "", err
	}
	for _, u := range list(out, "result") {
		if str(u, "username") == username {
			return str(u, "userId"), nil
		}
	}
	return "", nil
}
