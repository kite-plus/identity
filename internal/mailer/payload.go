// Package mailer renders the mail ZITADEL hands to its HTTP email provider
// and sends it over SMTP.
package mailer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Payload is what ZITADEL v4 posts for one email.
type Payload struct {
	ContextInfo struct {
		EventType string `json:"eventType"`
		Recipient string `json:"recipientEmailAddress"`
	} `json:"contextInfo"`
	TemplateData TemplateData   `json:"templateData"`
	Args         map[string]any `json:"args"`
}

// TemplateData holds the message texts, already localized and filled in.
type TemplateData struct {
	Title      string `json:"title"`
	PreHeader  string `json:"preHeader"`
	Subject    string `json:"subject"`
	Greeting   string `json:"greeting"`
	Text       string `json:"text"`
	URL        string `json:"url"`
	ButtonText string `json:"buttonText"`
	FooterText string `json:"footerText"`
}

// Code is the one-time code the mail carries, if any. ZITADEL lowercases
// only the first letter of each argument name, so "OTP" arrives as "oTP".
func (p Payload) Code() string {
	for _, k := range []string{"code", "oTP", "otp"} {
		if s, ok := p.Args[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// Signature errors.
var (
	ErrUnsigned     = errors.New("missing ZITADEL-Signature")
	ErrBadSignature = errors.New("signature does not match")
	ErrStale        = errors.New("signature timestamp out of tolerance")
)

// Verify checks a ZITADEL-Signature header ("t=<unix>,v1=<hex>") against the
// body: v1 is HMAC-SHA256 over "<unix>." + body with the provider's signing
// key, the same scheme as ZITADEL's pkg/actions.
func Verify(header string, body []byte, key string, now time.Time, tolerance time.Duration) error {
	if header == "" {
		return ErrUnsigned
	}
	var ts int64
	var sigs [][]byte
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("signature timestamp: %w", err)
			}
			ts = n
		case "v1":
			if b, err := hex.DecodeString(v); err == nil {
				sigs = append(sigs, b)
			}
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return ErrBadSignature
	}
	if d := now.Sub(time.Unix(ts, 0)); d > tolerance || d < -tolerance {
		return ErrStale
	}
	mac := hmac.New(sha256.New, []byte(key))
	fmt.Fprintf(mac, "%d.", ts)
	mac.Write(body)
	want := mac.Sum(nil)
	for _, s := range sigs {
		if hmac.Equal(want, s) {
			return nil
		}
	}
	return ErrBadSignature
}
