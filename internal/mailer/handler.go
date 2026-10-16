package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// Sender delivers a rendered mail.
type Sender interface {
	Send(ctx context.Context, m Mail) error
}

// Handler receives ZITADEL's HTTP email provider calls.
type Handler struct {
	// KeyFile holds the provider's signing key, written by identity-init.
	// It is read per request so a rotated key needs no restart.
	KeyFile string
	Site    Site
	Sender  Sender
	Log     *slog.Logger
	Now     func() time.Time
}

// ZITADEL gives up on a provider call after five seconds and retries, so a
// send that cannot finish sooner is better abandoned than sent twice.
const sendBudget = 4500 * time.Millisecond

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	key, err := os.ReadFile(h.KeyFile)
	if err != nil || len(strings.TrimSpace(string(key))) == 0 {
		h.Log.Error("signing key unavailable", "file", h.KeyFile, "error", err)
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	if err := Verify(r.Header.Get("ZITADEL-Signature"), body, strings.TrimSpace(string(key)), now(), 5*time.Minute); err != nil {
		h.Log.Warn("rejected unsigned call", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var p Payload
	if err := json.Unmarshal(body, &p); err != nil || p.ContextInfo.Recipient == "" || p.TemplateData.Subject == "" {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	m, err := Render(p, h.Site)
	if err != nil {
		h.Log.Error("render failed", "event", p.ContextInfo.EventType, "error", err)
		http.Error(w, "render failed", http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), sendBudget)
	defer cancel()
	if err := h.Sender.Send(ctx, m); err != nil {
		h.Log.Error("send failed", "event", p.ContextInfo.EventType, "error", err)
		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		http.Error(w, "send failed", status)
		return
	}
	h.Log.Info("sent", "event", p.ContextInfo.EventType)
	w.WriteHeader(http.StatusNoContent)
}
