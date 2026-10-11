// Command identity-init applies config/ to a ZITADEL instance and writes
// the per-service env files. It is safe to run repeatedly.
//
//	identity-init -config config/dev.yaml -token-file dev/out/bootstrap/admin.pat -out dev/out
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/kite-plus/identity/internal/apply"
	"github.com/kite-plus/identity/internal/config"
	"github.com/kite-plus/identity/internal/zitadel"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "identity-init:", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := flag.String("config", "", "configuration file, e.g. config/dev.yaml")
	tokenFile := flag.String("token-file", "", "file holding an IAM_OWNER personal access token (default $ZITADEL_TOKEN)")
	outDir := flag.String("out", "", "directory for state.json and the env files")
	apiURL := flag.String("api", "", "where to send requests (default: the issuer); the Host header stays the issuer's host")
	wait := flag.Duration("wait", 2*time.Minute, "how long to wait for the instance to become ready")
	only := flag.String("only", "", `run a single step: "readers" grants new readers the self-deletion role`)
	flag.Parse()
	if *cfgPath == "" || (*outDir == "" && *only == "") {
		flag.Usage()
		return fmt.Errorf("-config and -out are required")
	}
	if *only != "" && *only != "readers" {
		return fmt.Errorf("unknown step %q", *only)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	token := os.Getenv("ZITADEL_TOKEN")
	if *tokenFile != "" {
		b, err := os.ReadFile(*tokenFile)
		if err != nil {
			return err
		}
		token = string(b)
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("no token: pass -token-file or set ZITADEL_TOKEN")
	}
	issuer, err := url.Parse(cfg.Issuer)
	if err != nil {
		return err
	}
	base := cfg.Issuer
	if *apiURL != "" {
		base = *apiURL
	}
	client := zitadel.New(base, issuer.Host, token)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := waitReady(ctx, client, *wait); err != nil {
		return err
	}

	applier := apply.New(cfg, client, *outDir, os.Stdout)
	if *only == "readers" {
		n, err := applier.RunReaders(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("%d readers updated\n", n)
		return nil
	}
	fmt.Printf("applying %s to %s\n", *cfgPath, cfg.Issuer)
	n, err := applier.Run(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		fmt.Println("no changes")
	} else {
		fmt.Printf("%d changes\n", n)
	}
	fmt.Printf("env files in %s\n", *outDir)
	return nil
}

func waitReady(ctx context.Context, c *zitadel.Client, d time.Duration) error {
	deadline := time.Now().Add(d)
	for {
		err := c.Call(ctx, "GET", "/admin/v1/instances/me", "", nil, nil)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("instance not ready: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
