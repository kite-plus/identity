package apply

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// State keeps what ZITADEL shows only once (client secrets, tokens) and
// hashes of write-only values, so a re-run can tell whether anything needs
// to change. It lives next to the generated env files and is never
// committed.
type State struct {
	InstanceID string                  `json:"instanceId"`
	Apps       map[string]AppState     `json:"apps"`
	Machines   map[string]MachineState `json:"machines"`
	// Hashes of write-only values, keyed by what they belong to.
	Hashes map[string]string `json:"hashes"`
}

type AppState struct {
	AppID        string `json:"appId"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type MachineState struct {
	UserID string `json:"userId"`
	PATID  string `json:"patId"`
	Token  string `json:"token"`
}

func loadState(path string) (*State, error) {
	s := &State{}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, s); err != nil {
			return nil, err
		}
	}
	s.init()
	return s, nil
}

func (s *State) init() {
	if s.Apps == nil {
		s.Apps = map[string]AppState{}
	}
	if s.Machines == nil {
		s.Machines = map[string]MachineState{}
	}
	if s.Hashes == nil {
		s.Hashes = map[string]string{}
	}
}

// reset forgets everything; used when the state belongs to another
// instance, e.g. after `make dev-reset`.
func (s *State) reset(instanceID string) {
	*s = State{InstanceID: instanceID}
	s.init()
}

func (s *State) save(path string) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeSecret(path, append(b, '\n'))
}

func hash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeSecret(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
