package apply

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

func (a *Applier) smtp(ctx context.Context) error {
	c := a.cfg.SMTP
	if c.Host == "" {
		a.note("smtp not configured, skipped")
		return nil
	}
	var cur obj
	if err := a.call(ctx, http.MethodPost, "/admin/v1/email/_search", "", obj{}, &cur); err != nil {
		return err
	}
	want := obj{
		"description":    c.Description,
		"senderAddress":  c.SenderAddress,
		"senderName":     c.SenderName,
		"tls":            c.TLS,
		"host":           c.Host,
		"user":           c.User,
		"replyToAddress": c.ReplyTo,
	}
	var id, state string
	var have obj
	for _, p := range list(cur, "result") {
		if str(p, "description") == c.Description {
			id, state, have = str(p, "id"), str(p, "state"), sub(p, "smtp")
			have["description"] = p["description"]
			break
		}
	}
	pwHash := hash("smtp", c.Host, c.User, c.Password)
	if id == "" {
		in := merged(want, obj{"password": c.Password})
		var out obj
		if err := a.call(ctx, http.MethodPost, "/admin/v1/email/smtp", "", in, &out); err != nil {
			return err
		}
		id = str(out, "id")
		a.state.Hashes["smtp"] = pwHash
		a.changed("smtp provider %q created", c.Description)
	} else {
		if d := diff(want, have); len(d) > 0 {
			if err := a.call(ctx, http.MethodPut, "/admin/v1/email/smtp/"+id, "", want, nil); err != nil {
				return err
			}
			a.changed("smtp provider %v", d)
		}
		if a.state.Hashes["smtp"] != pwHash {
			if err := a.call(ctx, http.MethodPut, "/admin/v1/email/smtp/"+id+"/password", "", obj{"password": c.Password}, nil); err != nil {
				return err
			}
			a.state.Hashes["smtp"] = pwHash
			a.changed("smtp password")
		}
	}
	if state != "EMAIL_PROVIDER_ACTIVE" {
		if err := a.call(ctx, http.MethodPost, "/admin/v1/email/"+id+"/_activate", "", obj{}, nil); err != nil {
			return err
		}
		a.changed("smtp provider activated")
	}
	return nil
}

// merged returns a merged copy; later maps win.
func merged(ms ...obj) obj {
	out := obj{}
	for _, m := range ms {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

type asset struct {
	file      string
	urlField  string
	uploadURL string
}

func (a *Applier) branding(ctx context.Context) error {
	b := a.cfg.Branding
	want := obj{
		"primaryColor":        b.PrimaryColor,
		"backgroundColor":     b.BackgroundColor,
		"warnColor":           b.WarnColor,
		"fontColor":           b.FontColor,
		"primaryColorDark":    b.PrimaryColorDark,
		"backgroundColorDark": b.BackgroundColorDark,
		"warnColorDark":       b.WarnColorDark,
		"fontColorDark":       b.FontColorDark,
		"hideLoginNameSuffix": b.HideLoginNameSuffix,
		"disableWatermark":    b.DisableWatermark,
		"themeMode":           "THEME_MODE_AUTO",
	}
	var preview obj
	if err := a.call(ctx, http.MethodGet, "/admin/v1/policies/label/_preview", "", nil, &preview); err != nil {
		return err
	}
	pp := sub(preview, "policy")
	dirty := false
	if d := diff(want, pp); len(d) > 0 {
		if err := a.call(ctx, http.MethodPut, "/admin/v1/policies/label", "", want, nil); err != nil {
			return err
		}
		a.changed("branding %v", d)
		dirty = true
	}
	assets := []asset{
		{b.Logo, "logoUrl", "/assets/v1/instance/policy/label/logo"},
		{b.LogoDark, "logoUrlDark", "/assets/v1/instance/policy/label/logo/dark"},
		{b.Icon, "iconUrl", "/assets/v1/instance/policy/label/icon"},
		{b.IconDark, "iconUrlDark", "/assets/v1/instance/policy/label/icon/dark"},
	}
	for _, as := range assets {
		if as.file == "" {
			continue
		}
		path := a.cfg.Path(as.file)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if u := str(pp, as.urlField); u != "" {
			if cur, err := a.api.Fetch(ctx, u); err == nil && bytes.Equal(cur, data) {
				continue
			}
		}
		ct := mime.TypeByExtension(filepath.Ext(path))
		if err := a.api.Upload(ctx, as.uploadURL, filepath.Base(path), ct, data); err != nil {
			return fmt.Errorf("upload %s: %w", as.file, err)
		}
		a.changed("branding asset %s", as.file)
		dirty = true
	}
	if !dirty {
		var active obj
		if err := a.call(ctx, http.MethodGet, "/admin/v1/policies/label", "", nil, &active); err != nil {
			return err
		}
		ap := sub(active, "policy")
		if len(diff(want, ap)) > 0 {
			dirty = true
		}
		for _, as := range assets {
			if as.file != "" && str(ap, as.urlField) != str(pp, as.urlField) {
				dirty = true
			}
		}
	}
	if dirty {
		if err := a.call(ctx, http.MethodPost, "/admin/v1/policies/label/_activate", "", obj{}, nil); err != nil {
			return err
		}
		a.changed("branding activated")
	}
	return nil
}

func (a *Applier) messages(ctx context.Context) error {
	for _, m := range a.cfg.Messages {
		path := fmt.Sprintf("/admin/v1/text/message/%s/%s", m.Type, m.Language)
		want := obj{
			"title":      m.Title,
			"preHeader":  m.PreHeader,
			"subject":    m.Subject,
			"greeting":   m.Greeting,
			"text":       m.Text,
			"buttonText": m.ButtonText,
			"footerText": m.FooterText,
		}
		var cur obj
		if err := a.call(ctx, http.MethodGet, path, "", nil, &cur); err != nil {
			return err
		}
		ct := sub(cur, "customText")
		if ct["isDefault"] != true && len(diff(want, ct)) == 0 {
			continue
		}
		if err := a.call(ctx, http.MethodPut, path, "", want, nil); err != nil {
			return err
		}
		a.changed("message text %s/%s", m.Type, m.Language)
	}
	return nil
}
