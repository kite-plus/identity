package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"
)

// The images mail.html shows as cid:<name>@kite.plus travel inside the
// message, so opening it fetches nothing and no inbox blocks them.
//
//go:embed img/*.png
var images embed.FS

// SMTP sends through smtp://[user:pass@]host:port (STARTTLS when offered)
// or smtps://... (implicit TLS).
type SMTP struct {
	url      *url.URL
	from     *mail.Address
	replyTo  string
	hostname string
}

// NewSMTP checks the URL and the From address up front.
func NewSMTP(rawURL, from, replyTo, hostname string) (*SMTP, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "smtp" && u.Scheme != "smtps") || u.Hostname() == "" {
		return nil, errors.New("MAILER_SMTP_URL: not an smtp:// or smtps:// URL")
	}
	f, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("MAILER_FROM: %w", err)
	}
	if replyTo != "" {
		if _, err := mail.ParseAddress(replyTo); err != nil {
			return nil, fmt.Errorf("MAILER_REPLY_TO: %w", err)
		}
	}
	if hostname == "" {
		hostname = "localhost"
	}
	return &SMTP{url: u, from: f, replyTo: replyTo, hostname: hostname}, nil
}

// Send delivers one message.
func (s *SMTP) Send(ctx context.Context, m Mail) error {
	raw, err := s.compose(m)
	if err != nil {
		return err
	}
	host, port := s.url.Hostname(), s.url.Port()
	if port == "" {
		port = map[string]string{"smtp": "25", "smtps": "465"}[s.url.Scheme]
	}
	tlsConf := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	var conn net.Conn
	if s.url.Scheme == "smtps" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConf}).DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	}
	if err != nil {
		return err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.Hello(s.hostname); err != nil {
		return err
	}
	if s.url.Scheme == "smtp" {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConf); err != nil {
				return err
			}
		}
	}
	if u := s.url.User; u != nil {
		pass, _ := u.Password()
		if err := c.Auth(smtp.PlainAuth("", u.Username(), pass, host)); err != nil {
			return err
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(m.To); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (s *SMTP) compose(m Mail) ([]byte, error) {
	if strings.ContainsAny(m.To, "\r\n") {
		return nil, errors.New("bad recipient")
	}
	var b bytes.Buffer
	alt := multipart.NewWriter(&b)
	h := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	h("From", s.from.String())
	h("To", m.To)
	if s.replyTo != "" {
		h("Reply-To", s.replyTo)
	}
	h("Subject", mime.BEncoding.Encode("UTF-8", m.Subject))
	h("Date", time.Now().UTC().Format(time.RFC1123Z))
	h("Message-ID", "<"+randomHex(16)+"@"+s.hostname+">")
	h("MIME-Version", "1.0")
	h("Auto-Submitted", "auto-generated")
	h("Content-Type", `multipart/alternative; boundary="`+alt.Boundary()+`"`)
	b.WriteString("\r\n")
	if err := textPart(alt, "text/plain", m.Text); err != nil {
		return nil, err
	}
	relBoundary := randomHex(15)
	pw, err := alt.CreatePart(textproto.MIMEHeader{"Content-Type": {`multipart/related; type="text/html"; boundary="` + relBoundary + `"`}})
	if err != nil {
		return nil, err
	}
	rel := multipart.NewWriter(pw)
	if err := rel.SetBoundary(relBoundary); err != nil {
		return nil, err
	}
	if err := textPart(rel, "text/html", m.HTML); err != nil {
		return nil, err
	}
	if err := imageParts(rel); err != nil {
		return nil, err
	}
	if err := rel.Close(); err != nil {
		return nil, err
	}
	if err := alt.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func textPart(w *multipart.Writer, ct, body string) error {
	pw, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {ct + "; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(pw)
	if _, err := qp.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return err
	}
	return qp.Close()
}

func imageParts(w *multipart.Writer) error {
	names, err := images.ReadDir("img")
	if err != nil {
		return err
	}
	for _, n := range names {
		data, err := images.ReadFile("img/" + n.Name())
		if err != nil {
			return err
		}
		id := strings.TrimSuffix(n.Name(), path.Ext(n.Name()))
		pw, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {`image/png; name="` + n.Name() + `"`},
			"Content-Transfer-Encoding": {"base64"},
			"Content-ID":                {"<" + id + "@kite.plus>"},
			"Content-Disposition":       {`inline; filename="` + n.Name() + `"`},
		})
		if err != nil {
			return err
		}
		if err := writeBase64(pw, data); err != nil {
			return err
		}
	}
	return nil
}

// writeBase64 wraps at 76 columns as RFC 2045 requires.
func writeBase64(w io.Writer, data []byte) error {
	enc := base64.StdEncoding.EncodeToString(data)
	for len(enc) > 76 {
		if _, err := io.WriteString(w, enc[:76]+"\r\n"); err != nil {
			return err
		}
		enc = enc[76:]
	}
	_, err := io.WriteString(w, enc+"\r\n")
	return err
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
