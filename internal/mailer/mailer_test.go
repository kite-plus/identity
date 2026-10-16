package mailer

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const key = "test-signing-key"

func sign(t time.Time, body []byte, k string) string {
	mac := hmac.New(sha256.New, []byte(k))
	fmt.Fprintf(mac, "%d.", t.Unix())
	mac.Write(body)
	return fmt.Sprintf("t=%d,v1=%s", t.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func TestVerify(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	body := []byte(`{"a":1}`)
	if err := Verify(sign(now, body, key), body, key, now, time.Minute); err != nil {
		t.Fatalf("valid: %v", err)
	}
	// ZITADEL lists one v1 per active key; any match is enough.
	two := sign(now, body, "old") + ",v1=" + strings.Split(sign(now, body, key), "v1=")[1]
	if err := Verify(two, body, key, now, time.Minute); err != nil {
		t.Fatalf("rotated: %v", err)
	}
	for name, c := range map[string]struct {
		header string
		want   error
	}{
		"unsigned":  {"", ErrUnsigned},
		"wrong key": {sign(now, body, "other"), ErrBadSignature},
		"stale":     {sign(now.Add(-10*time.Minute), body, key), ErrStale},
		"no v1":     {fmt.Sprintf("t=%d", now.Unix()), ErrBadSignature},
	} {
		if err := Verify(c.header, body, key, now, time.Minute); err != c.want {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
	if err := Verify(sign(now, body, key), []byte(`{"a":2}`), key, now, time.Minute); err != ErrBadSignature {
		t.Errorf("tampered body: %v", err)
	}
}

var site = Site{URL: "https://id.kite.plus"}

func reset() Payload {
	var p Payload
	p.ContextInfo.EventType = "user.human.password.code.added"
	p.ContextInfo.Recipient = "reader@example.org"
	p.TemplateData = TemplateData{
		Title: "重置你的密码", PreHeader: "重置密码", Subject: "重置你的 Kite Plus 账号密码",
		Greeting: "你好，友人：", Text: "我们收到了重置密码的请求。\n点下面的按钮设置新密码。",
		URL:        "https://id.kite.plus/ui/v2/login/password/set?code=AG7889&userId=1&organization=2",
		ButtonText: "设置新密码", FooterText: "没有申请重置？忽略这封邮件即可。",
	}
	p.Args = map[string]any{"code": "AG7889"}
	return p
}

func TestRenderReset(t *testing.T) {
	m, err := Render(reset(), site)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<html lang="zh-CN">`, "重置你的密码", "我们收到了重置密码的请求。", "点下面的按钮设置新密码。",
		">AG7889<", "验证码", "设置新密码", "没有申请重置？",
		`href="https://id.kite.plus/ui/v2/login/password/set?code=AG7889&amp;userId=1&amp;organization=2"`,
		`src="cid:kite-tie@kite.plus"`, "Kite Plus 账号 · id.kite.plus",
		">验证码 AG7889<",
	} {
		if !strings.Contains(m.HTML, want) {
			t.Errorf("HTML lacks %q", want)
		}
	}
	if m.To != "reader@example.org" || m.Subject != "重置你的 Kite Plus 账号密码" {
		t.Errorf("headers: %+v", m)
	}
	for _, want := range []string{"验证码: AG7889", "设置新密码: https://id.kite.plus/ui/v2/login/password/set?code=AG7889&userId=1"} {
		if !strings.Contains(m.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, m.Text)
		}
	}
}

func TestRenderOTPWithoutButton(t *testing.T) {
	var p Payload
	p.ContextInfo.Recipient = "reader@example.org"
	p.TemplateData = TemplateData{Title: "Your sign-in code", Subject: "Your Kite Plus Account code", Text: "Use this code to sign in.", ButtonText: "Verify"}
	p.Args = map[string]any{"oTP": "482913"}
	m, err := Render(p, site)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.HTML, `<html lang="en">`) || !strings.Contains(m.HTML, ">482913<") || !strings.Contains(m.HTML, "One account for Explore") {
		t.Error("English OTP mail misses its code or texts")
	}
	if strings.Contains(m.HTML, "&rarr;") || strings.Contains(m.Text, "Verify:") {
		t.Error("a mail without a URL must have no button")
	}
}

func TestRenderEscapes(t *testing.T) {
	p := reset()
	p.TemplateData.Greeting = `<script>alert(1)</script>`
	p.TemplateData.URL = "javascript:alert(1)"
	m, err := Render(p, site)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") || strings.Contains(m.HTML, `href="javascript:`) {
		t.Error("unsafe content reached the HTML")
	}
}

type recorder struct{ sent []Mail }

func (r *recorder) Send(_ context.Context, m Mail) error { r.sent = append(r.sent, m); return nil }

func TestHandler(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "signing-key")
	rec := &recorder{}
	now := time.Unix(1_800_000_000, 0)
	h := &Handler{KeyFile: keyFile, Site: site, Sender: rec, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now }}
	body := []byte(`{"contextInfo":{"eventType":"user.human.password.code.added","recipientEmailAddress":"reader@example.org"},` +
		`"templateData":{"title":"重置你的密码","subject":"重置你的 Kite Plus 账号密码","text":"正文"},"args":{"code":"AG7889"}}`)
	call := func(sig string) int {
		req := httptest.NewRequest(http.MethodPost, "/zitadel/email", strings.NewReader(string(body)))
		if sig != "" {
			req.Header.Set("ZITADEL-Signature", sig)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	if c := call(sign(now, body, key)); c != http.StatusServiceUnavailable {
		t.Fatalf("without a key file: %d", c)
	}
	if err := os.WriteFile(keyFile, []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := call(sign(now, body, "other")); c != http.StatusUnauthorized {
		t.Fatalf("wrong signature: %d", c)
	}
	if c := call(sign(now, body, key)); c != http.StatusNoContent {
		t.Fatalf("signed call: %d", c)
	}
	if len(rec.sent) != 1 || rec.sent[0].To != "reader@example.org" || !strings.Contains(rec.sent[0].HTML, "AG7889") {
		t.Fatalf("sent: %+v", rec.sent)
	}
}

func TestCompose(t *testing.T) {
	s, err := NewSMTP("smtps://resend:key@smtp.resend.com:465", "Kite Plus 账号 <no-reply@id.kite.plus>", "support@kite.plus", "id.kite.plus")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := s.compose(Mail{To: "reader@example.org", Subject: "重置你的密码", Text: "t", HTML: "<p>h</p>"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"From: =?utf-8?q?Kite_Plus_", "<no-reply@id.kite.plus>", "Reply-To: support@kite.plus", "Subject: =?UTF-8?b?", "Message-ID: <", "@id.kite.plus>", "multipart/alternative", "multipart/related", "Content-ID: <kite-mark@kite.plus>", "Content-ID: <kite-tie@kite.plus>"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("message lacks %q", want)
		}
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	parts := 0
	r := multipart.NewReader(msg.Body, params["boundary"])
	for {
		if _, err := r.NextPart(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		parts++
	}
	if parts != 2 {
		t.Errorf("alternative has %d parts, want text and related", parts)
	}
	if _, err := s.compose(Mail{To: "a@example.org\r\nBcc: b@example.org"}); err == nil {
		t.Error("a recipient with a line break must be refused")
	}
	if _, err := NewSMTP("http://example.org", "a@example.org", "", ""); err == nil {
		t.Error("a non-smtp URL must be refused")
	}
}
