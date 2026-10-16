// Command identity-mailer is ZITADEL's HTTP email provider: it renders each
// account mail in the Kite Plus look and sends it over SMTP.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kite-plus/identity/internal/mailer"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the running server and exit")
	flag.Parse()
	listen := env("MAILER_LISTEN", ":8080")
	if *healthcheck {
		os.Exit(probe(listen))
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log, listen); err != nil {
		log.Error("exit", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, listen string) error {
	siteURL := os.Getenv("MAILER_SITE_URL")
	if siteURL == "" {
		return errors.New("MAILER_SITE_URL is required")
	}
	sender, err := mailer.NewSMTP(os.Getenv("MAILER_SMTP_URL"), os.Getenv("MAILER_FROM"), os.Getenv("MAILER_REPLY_TO"), os.Getenv("MAILER_HOSTNAME"))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/zitadel/email", &mailer.Handler{
		KeyFile: env("MAILER_SIGNING_KEY_FILE", "/secrets/signing-key"),
		Site:    mailer.Site{URL: siteURL},
		Sender:  sender,
		Log:     log,
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	srv := &http.Server{Addr: listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
	}()
	log.Info("listening", "addr", listen)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func probe(listen string) int {
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return 1
	}
	return 0
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
