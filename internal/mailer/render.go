package mailer

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/url"
	"strings"
	"unicode"
)

//go:embed mail.html
var mailHTML string

var page = template.Must(template.New("mail").Parse(mailHTML))

// Site describes the service the mail comes from.
type Site struct {
	// URL is the public origin, such as https://id.kite.plus.
	URL string
}

// Mail is a rendered message.
type Mail struct {
	To, Subject, Text, HTML string
}

type view struct {
	Lang, Subject, Preview, Eyebrow, Heading, Greeting string
	Paragraphs                                         []string
	Code, CodeLabel, ButtonText, URL, FallbackLabel    string
	Note, Service, Site, SiteURL, Tagline              string
}

// Render lays a payload out in the Kite Plus paper look. The message texts
// map onto it as: preHeader the small label, title the heading, text the
// body, footerText the note under the dashed rule.
func Render(p Payload, s Site) (Mail, error) {
	d := p.TemplateData
	zh := hasHan(d.Subject + d.Title + d.Text)
	v := view{
		Lang:          pick(zh, "zh-CN", "en"),
		Subject:       d.Subject,
		Eyebrow:       d.PreHeader,
		Heading:       firstOf(d.Title, d.Subject),
		Greeting:      d.Greeting,
		Paragraphs:    lines(d.Text),
		Code:          p.Code(),
		CodeLabel:     pick(zh, "验证码", "Code"),
		FallbackLabel: pick(zh, "按钮打不开？把这个链接复制到浏览器：", "Button not working? Copy this link into your browser:"),
		Note:          d.FooterText,
		Service:       pick(zh, "账号", "Account"),
		SiteURL:       strings.TrimRight(s.URL, "/"),
		Tagline:       pick(zh, "一个账号，通行 Explore、Kite 评论和 Kite 应用", "One account for Explore, Kite Comments and the Kite app"),
	}
	if u, err := url.Parse(v.SiteURL); err == nil {
		v.Site = u.Host
	}
	if d.URL != "" && d.ButtonText != "" {
		v.URL, v.ButtonText = d.URL, d.ButtonText
	}
	v.Preview = preview(v)

	var h bytes.Buffer
	if err := page.Execute(&h, v); err != nil {
		return Mail{}, err
	}
	return Mail{To: p.ContextInfo.Recipient, Subject: d.Subject, Text: text(v), HTML: h.String()}, nil
}

// preview is the line inboxes show after the subject: the code when there
// is one, else the start of the body.
func preview(v view) string {
	if v.Code != "" {
		return v.CodeLabel + " " + v.Code
	}
	if len(v.Paragraphs) > 0 {
		return v.Paragraphs[0]
	}
	return v.Heading
}

func text(v view) string {
	var b strings.Builder
	add := func(s string) {
		if s != "" {
			b.WriteString(s + "\n\n")
		}
	}
	add(v.Heading)
	add(v.Greeting)
	for _, p := range v.Paragraphs {
		add(p)
	}
	if v.Code != "" {
		add(v.CodeLabel + ": " + v.Code)
	}
	if v.URL != "" {
		add(v.ButtonText + ": " + v.URL)
	}
	add(v.Note)
	b.WriteString("--\nKite Plus " + v.Service + " · " + v.Site + "\n")
	return b.String()
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func pick(zh bool, a, b string) string {
	if zh {
		return a
	}
	return b
}

func firstOf(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
