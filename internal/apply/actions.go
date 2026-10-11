package apply

import (
	"context"
	"net/http"
	"slices"
)

const targetTimeout = "10s"

func targetName(appKey string) string { return appKey + "-user-events" }

// actions creates one Actions v2 target per app and one execution per
// configured user event, calling every app's target.
func (a *Applier) actions(ctx context.Context) error {
	var out obj
	if err := a.call(ctx, http.MethodPost, "/v2/actions/targets/search", "", obj{}, &out); err != nil {
		return err
	}
	byName := map[string]obj{}
	for _, t := range list(out, "targets") {
		byName[str(t, "name")] = t
	}
	var ids []string
	for _, app := range a.cfg.Apps {
		if app.WebhookURL == "" {
			continue
		}
		name := targetName(app.Key)
		want := obj{
			"name":        name,
			"restWebhook": obj{"interruptOnError": false},
			"timeout":     targetTimeout,
			"endpoint":    app.WebhookURL,
			"payloadType": "PAYLOAD_TYPE_JSON",
		}
		cur, ok := byName[name]
		switch {
		case !ok:
			var created obj
			if err := a.call(ctx, http.MethodPost, "/v2/actions/targets", "", want, &created); err != nil {
				return err
			}
			ids = append(ids, str(created, "id"))
			a.changed("target %s created", name)
		default:
			id := str(cur, "id")
			if _, isWebhook := cur["restWebhook"]; !isWebhook || len(diff(want, cur)) > 0 {
				if err := a.call(ctx, http.MethodPost, "/v2/actions/targets/"+id, "", want, nil); err != nil {
					return err
				}
				a.changed("target %s %v", name, diff(want, cur))
			}
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)

	var execs obj
	if err := a.call(ctx, http.MethodPost, "/v2/actions/executions/search", "", obj{}, &execs); err != nil {
		return err
	}
	current := map[string][]string{}
	for _, e := range list(execs, "executions") {
		if ev := str(e, "condition", "event", "event"); ev != "" {
			current[ev] = strs(e["targets"])
		}
	}
	for _, ev := range a.cfg.Events {
		have := slices.Clone(current[ev])
		slices.Sort(have)
		if slices.Equal(have, ids) {
			continue
		}
		in := obj{"condition": obj{"event": obj{"event": ev}}, "targets": ids}
		if err := a.call(ctx, http.MethodPut, "/v2/actions/executions", "", in, nil); err != nil {
			return err
		}
		a.changed("execution on %s", ev)
	}
	for ev, targets := range current {
		if slices.Contains(a.cfg.Events, ev) {
			continue
		}
		rest := slices.DeleteFunc(slices.Clone(targets), func(t string) bool { return slices.Contains(ids, t) })
		if len(rest) == len(targets) {
			continue
		}
		in := obj{"condition": obj{"event": obj{"event": ev}}, "targets": rest}
		if err := a.call(ctx, http.MethodPut, "/v2/actions/executions", "", in, nil); err != nil {
			return err
		}
		a.changed("execution on %s no longer calls the app targets", ev)
	}
	return nil
}

// signingKey returns the HMAC key of an app's target.
func (a *Applier) signingKey(ctx context.Context, appKey string) (string, error) {
	var out obj
	if err := a.call(ctx, http.MethodPost, "/v2/actions/targets/search", "", obj{}, &out); err != nil {
		return "", err
	}
	for _, t := range list(out, "targets") {
		if str(t, "name") == targetName(appKey) {
			return str(t, "signingKey"), nil
		}
	}
	return "", nil
}
