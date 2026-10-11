package apply

import (
	"context"
	"fmt"
	"net/http"
	"slices"
)

// selfManagerRole carries only policy.read and user.self.delete. ZITADEL
// gives it to nobody by default, and without it a reader cannot delete
// their own account (R6), neither in the console nor through the API.
const selfManagerRole = "ORG_USER_SELF_MANAGER"

// ReaderSelfService grants selfManagerRole to every human in the readers
// org that lacks it. New sign-ups do not get it automatically, so this step
// also runs on its own on a schedule (identity-init -only readers).
func (a *Applier) ReaderSelfService(ctx context.Context) error {
	if !a.cfg.ReaderSelfDeletion {
		return nil
	}
	if a.readersID == "" {
		id, err := a.findOrg(ctx, a.cfg.Orgs.Readers)
		if err != nil {
			return err
		}
		if id == "" {
			return fmt.Errorf("readers org %q not found", a.cfg.Orgs.Readers)
		}
		a.readersID = id
	}

	members := map[string][]string{}
	for offset := 0; ; offset += pageSize {
		var out obj
		in := obj{"query": obj{"offset": fmt.Sprint(offset), "limit": pageSize}}
		if err := a.call(ctx, http.MethodPost, "/management/v1/orgs/me/members/_search", a.readersID, in, &out); err != nil {
			return err
		}
		page := list(out, "result")
		for _, m := range page {
			members[str(m, "userId")] = strs(m["roles"])
		}
		if len(page) < pageSize {
			break
		}
	}

	for offset := 0; ; offset += pageSize {
		var out obj
		in := obj{
			"query": obj{"offset": fmt.Sprint(offset), "limit": pageSize, "asc": true},
			"queries": []obj{
				{"organizationIdQuery": obj{"organizationId": a.readersID}},
				{"typeQuery": obj{"type": "TYPE_HUMAN"}},
			},
		}
		if err := a.call(ctx, http.MethodPost, "/v2/users", "", in, &out); err != nil {
			return err
		}
		page := list(out, "result")
		for _, u := range page {
			id := str(u, "userId")
			roles, isMember := members[id]
			switch {
			case slices.Contains(roles, selfManagerRole):
				continue
			case isMember:
				in := obj{"roles": append(slices.Clone(roles), selfManagerRole)}
				if err := a.call(ctx, http.MethodPut, "/management/v1/orgs/me/members/"+id, a.readersID, in, nil); err != nil {
					return err
				}
			default:
				in := obj{"userId": id, "roles": []string{selfManagerRole}}
				if err := a.call(ctx, http.MethodPost, "/management/v1/orgs/me/members", a.readersID, in, nil); err != nil {
					return err
				}
			}
			a.changed("reader %s may delete their own account", str(u, "username"))
		}
		if len(page) < pageSize {
			break
		}
	}
	return nil
}

const pageSize = 500
