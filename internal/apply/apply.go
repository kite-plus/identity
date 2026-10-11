// Package apply makes a ZITADEL instance match config/. Every step reads
// the current value first and writes only on a difference, so running it
// twice in a row changes nothing the second time.
package apply

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/kite-plus/identity/internal/config"
	"github.com/kite-plus/identity/internal/zitadel"
)

type Applier struct {
	cfg    *config.Config
	api    *zitadel.Client
	outDir string
	log    io.Writer
	state  *State

	instanceID  string
	operatorsID string
	readersID   string

	changes int
}

func New(cfg *config.Config, api *zitadel.Client, outDir string, log io.Writer) *Applier {
	return &Applier{cfg: cfg, api: api, outDir: outDir, log: log}
}

// Run applies the whole configuration and writes the env files. It returns
// the number of changes made.
func (a *Applier) Run(ctx context.Context) (int, error) {
	statePath := filepath.Join(a.outDir, "state.json")
	st, err := loadState(statePath)
	if err != nil {
		return 0, fmt.Errorf("load state: %w", err)
	}
	a.state = st

	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"instance", a.instance},
		{"organizations", a.orgs},
		{"login policy", a.loginPolicy},
		{"operator login policy", a.operatorLoginPolicy},
		{"lockout and password policy", a.passwordPolicies},
		{"privacy policy", a.privacyPolicy},
		{"smtp", a.smtp},
		{"branding", a.branding},
		{"message texts", a.messages},
		{"login texts", a.loginTexts},
		{"identity providers", a.idps},
		{"projects and apps", a.projectAndApps},
		{"event targets", a.actions},
		{"reader language hook", a.readerLanguage},
		{"sync machine users", a.machines},
		{"test users", a.testUsers},
		{"reader self-service", a.ReaderSelfService},
	}
	for _, s := range steps {
		if err := s.fn(ctx); err != nil {
			// Keep secrets learned so far; they cannot be read again.
			_ = a.state.save(statePath)
			return a.changes, fmt.Errorf("%s: %w", s.name, err)
		}
	}
	if err := a.state.save(statePath); err != nil {
		return a.changes, err
	}
	if err := a.writeOutputs(ctx); err != nil {
		return a.changes, err
	}
	return a.changes, nil
}

// RunReaders applies only the reader self-service step, for running on a
// schedule between full runs.
func (a *Applier) RunReaders(ctx context.Context) (int, error) {
	err := a.ReaderSelfService(ctx)
	return a.changes, err
}

func (a *Applier) changed(format string, args ...any) {
	a.changes++
	fmt.Fprintf(a.log, "  changed: "+format+"\n", args...)
}

func (a *Applier) note(format string, args ...any) {
	fmt.Fprintf(a.log, "  note: "+format+"\n", args...)
}

// call is Client.Call that treats "nothing changed" as success.
func (a *Applier) call(ctx context.Context, method, path, orgID string, in, out any) error {
	err := a.api.Call(ctx, method, path, orgID, in, out)
	if zitadel.IsNoChange(err) {
		return nil
	}
	return err
}

const textEquals = "TEXT_QUERY_METHOD_EQUALS"

func isNotFound(err error) bool { return zitadel.IsNotFound(err) }
