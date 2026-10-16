package apply

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

// mailer points ZITADEL's email at identity-mailer and leaves the provider's
// signing key in <out>/mailer/signing-key for that service to read.
func (a *Applier) mailer(ctx context.Context) error {
	c := a.cfg.Mailer
	if c.Endpoint == "" {
		return nil
	}
	var cur obj
	if err := a.call(ctx, http.MethodPost, "/admin/v1/email/_search", "", obj{}, &cur); err != nil {
		return err
	}
	want := obj{"endpoint": c.Endpoint, "description": c.Description}
	var id, state, key string
	for _, p := range list(cur, "result") {
		h := sub(p, "http")
		if str(p, "description") != c.Description || h == nil {
			continue
		}
		id, state, key = str(p, "id"), str(p, "state"), str(h, "signingKey")
		if str(h, "endpoint") != c.Endpoint {
			if err := a.call(ctx, http.MethodPut, "/admin/v1/email/http/"+id, "", want, nil); err != nil {
				return err
			}
			a.changed("mailer endpoint set to %s", c.Endpoint)
		}
		break
	}
	if id == "" {
		var out obj
		if err := a.call(ctx, http.MethodPost, "/admin/v1/email/http", "", want, &out); err != nil {
			return err
		}
		id, key = str(out, "id"), str(out, "signingKey")
		a.changed("mailer provider %q created", c.Description)
	}
	if state != "EMAIL_PROVIDER_ACTIVE" {
		if err := a.call(ctx, http.MethodPost, "/admin/v1/email/"+id+"/_activate", "", obj{}, nil); err != nil {
			return err
		}
		a.changed("mailer provider activated")
	}
	if key == "" {
		return errors.New("ZITADEL returned no signing key for the mailer provider")
	}
	path := filepath.Join(a.outDir, "mailer", "signing-key")
	if have, err := os.ReadFile(path); err == nil && bytes.Equal(bytes.TrimSpace(have), []byte(key)) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(key+"\n"), 0o600); err != nil {
		return err
	}
	a.changed("mailer signing key written to %s", path)
	return nil
}
