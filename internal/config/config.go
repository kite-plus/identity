// Package config loads the declarative instance configuration in config/.
//
// A file may name a base file with "extends"; maps are merged key by key
// and the extending file wins. Values may reference environment variables
// as ${NAME} or ${NAME:-default}.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Environment names the output files and appears in reports.
	Environment string `yaml:"environment"`
	Issuer      string `yaml:"issuer"`

	Instance           Instance           `yaml:"instance"`
	Orgs               Orgs               `yaml:"orgs"`
	LoginPolicy        LoginPolicy        `yaml:"loginPolicy"`
	OperatorPolicy     OperatorPolicy     `yaml:"operatorLoginPolicy"`
	Lockout            Lockout            `yaml:"lockout"`
	PasswordComplexity PasswordComplexity `yaml:"passwordComplexity"`
	Privacy            Privacy            `yaml:"privacy"`
	SMTP               SMTP               `yaml:"smtp"`
	Branding           Branding           `yaml:"branding"`
	Messages           []Message          `yaml:"messages"`
	// LoginTexts overrides login UI strings per locale, keyed like the
	// login app's locales/*.json.
	LoginTexts map[string]map[string]any `yaml:"loginTexts"`
	Apps               []App              `yaml:"apps"`
	Events             []string           `yaml:"events"`
	SyncRole           string             `yaml:"syncRole"`
	SyncTokenLifetime  Duration           `yaml:"syncTokenLifetime"`
	IdPs               IdPs               `yaml:"idps"`
	TestUsers          []TestUser         `yaml:"testUsers"`

	// dir is the directory of the loaded file; asset paths are relative to
	// the repository root, which is its parent.
	dir string
}

type Instance struct {
	DefaultLanguage  string   `yaml:"defaultLanguage"`
	AllowedLanguages []string `yaml:"allowedLanguages"`
}

type Orgs struct {
	// Operators is the org created by ZITADEL setup; it holds admins and
	// machine users.
	Operators string `yaml:"operators"`
	// Readers becomes the default org, so self-registration lands there.
	Readers string `yaml:"readers"`
}

type LoginPolicy struct {
	AllowUsernamePassword      bool     `yaml:"allowUsernamePassword"`
	AllowRegister              bool     `yaml:"allowRegister"`
	AllowExternalIdP           bool     `yaml:"allowExternalIdp"`
	ForceMFA                   bool     `yaml:"forceMfa"`
	Passkeys                   bool     `yaml:"passkeys"`
	HidePasswordReset          bool     `yaml:"hidePasswordReset"`
	IgnoreUnknownUsernames     bool     `yaml:"ignoreUnknownUsernames"`
	AllowDomainDiscovery       bool     `yaml:"allowDomainDiscovery"`
	DisableLoginWithEmail      bool     `yaml:"disableLoginWithEmail"`
	DisableLoginWithPhone      bool     `yaml:"disableLoginWithPhone"`
	PasswordCheckLifetime      Duration `yaml:"passwordCheckLifetime"`
	ExternalLoginCheckLifetime Duration `yaml:"externalLoginCheckLifetime"`
	MFAInitSkipLifetime        Duration `yaml:"mfaInitSkipLifetime"`
	SecondFactorCheckLifetime  Duration `yaml:"secondFactorCheckLifetime"`
	MultiFactorCheckLifetime   Duration `yaml:"multiFactorCheckLifetime"`
	SecondFactors              []string `yaml:"secondFactors"`
	MultiFactors               []string `yaml:"multiFactors"`
}

type OperatorPolicy struct {
	ForceMFA                 bool     `yaml:"forceMfa"`
	PasswordCheckLifetime    Duration `yaml:"passwordCheckLifetime"`
	MultiFactorCheckLifetime Duration `yaml:"multiFactorCheckLifetime"`
}

type Lockout struct {
	MaxPasswordAttempts uint64 `yaml:"maxPasswordAttempts"`
	MaxOTPAttempts      uint64 `yaml:"maxOtpAttempts"`
}

type PasswordComplexity struct {
	MinLength    uint64 `yaml:"minLength"`
	HasUppercase bool   `yaml:"hasUppercase"`
	HasLowercase bool   `yaml:"hasLowercase"`
	HasNumber    bool   `yaml:"hasNumber"`
	HasSymbol    bool   `yaml:"hasSymbol"`
}

type Privacy struct {
	TOSLink      string `yaml:"tosLink"`
	PrivacyLink  string `yaml:"privacyLink"`
	HelpLink     string `yaml:"helpLink"`
	SupportEmail string `yaml:"supportEmail"`
	DocsLink     string `yaml:"docsLink"`
}

type SMTP struct {
	Description   string `yaml:"description"`
	Host          string `yaml:"host"`
	TLS           bool   `yaml:"tls"`
	User          string `yaml:"user"`
	Password      string `yaml:"password"`
	SenderAddress string `yaml:"senderAddress"`
	SenderName    string `yaml:"senderName"`
	ReplyTo       string `yaml:"replyTo"`
}

type Branding struct {
	PrimaryColor        string `yaml:"primaryColor"`
	BackgroundColor     string `yaml:"backgroundColor"`
	WarnColor           string `yaml:"warnColor"`
	FontColor           string `yaml:"fontColor"`
	PrimaryColorDark    string `yaml:"primaryColorDark"`
	BackgroundColorDark string `yaml:"backgroundColorDark"`
	WarnColorDark       string `yaml:"warnColorDark"`
	FontColorDark       string `yaml:"fontColorDark"`
	HideLoginNameSuffix bool   `yaml:"hideLoginNameSuffix"`
	DisableWatermark    bool   `yaml:"disableWatermark"`
	Logo                string `yaml:"logo"`
	LogoDark            string `yaml:"logoDark"`
	Icon                string `yaml:"icon"`
	IconDark            string `yaml:"iconDark"`
}

// Message overrides the text of one notification type in one language.
type Message struct {
	Type       string `yaml:"type"`
	Language   string `yaml:"language"`
	Title      string `yaml:"title"`
	PreHeader  string `yaml:"preHeader"`
	Subject    string `yaml:"subject"`
	Greeting   string `yaml:"greeting"`
	Text       string `yaml:"text"`
	ButtonText string `yaml:"buttonText"`
	FooterText string `yaml:"footerText"`
}

// App is one confidential OIDC client and the service behind it.
type App struct {
	// Key is the service name; it prefixes the variables in the env file.
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
	// Project defaults to Name. Each app gets its own project because
	// ZITADEL puts every client ID of a project into the ID token audience.
	Project                string   `yaml:"project"`
	RedirectURIs           []string `yaml:"redirectUris"`
	PostLogoutRedirectURIs []string `yaml:"postLogoutRedirectUris"`
	BackChannelLogoutURI   string   `yaml:"backChannelLogoutUri"`
	WebhookURL             string   `yaml:"webhookUrl"`
	// DevMode allows plain http redirect URIs; only for local development.
	DevMode bool `yaml:"devMode"`
}

type IdPs struct {
	GitHub GitHub `yaml:"github"`
	Apple  Apple  `yaml:"apple"`
}

type GitHub struct {
	Name         string   `yaml:"name"`
	ClientID     string   `yaml:"clientId"`
	ClientSecret string   `yaml:"clientSecret"`
	Scopes       []string `yaml:"scopes"`
}

func (g GitHub) Enabled() bool { return g.ClientID != "" && g.ClientSecret != "" }

type Apple struct {
	Name           string   `yaml:"name"`
	ClientID       string   `yaml:"clientId"`
	TeamID         string   `yaml:"teamId"`
	KeyID          string   `yaml:"keyId"`
	PrivateKeyFile string   `yaml:"privateKeyFile"`
	Scopes         []string `yaml:"scopes"`
}

func (a Apple) Enabled() bool {
	return a.ClientID != "" && a.TeamID != "" && a.KeyID != "" && a.PrivateKeyFile != ""
}

type TestUser struct {
	Username   string `yaml:"username"`
	GivenName  string `yaml:"givenName"`
	FamilyName string `yaml:"familyName"`
	Email      string `yaml:"email"`
	Password   string `yaml:"password"`
	Language   string `yaml:"language"`
}

// Duration accepts Go duration strings such as "720h".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	v, err := time.ParseDuration(n.Value)
	if err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}
	*d = Duration(v)
	return nil
}

func (d Duration) Std() time.Duration { return time.Duration(d) }

// Path resolves a repository-relative path such as branding/logo.svg.
func (c *Config) Path(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.dir, "..", p)
}

// Load reads path, follows "extends" and expands environment variables.
func Load(path string) (*Config, error) {
	raw, err := loadMerged(path, 0)
	if err != nil {
		return nil, err
	}
	delete(raw, "extends")
	expanded := expandAll(raw)
	b, err := yaml.Marshal(expanded)
	if err != nil {
		return nil, err
	}
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	c.dir = filepath.Dir(path)
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

func loadMerged(path string, depth int) (map[string]any, error) {
	if depth > 5 {
		return nil, fmt.Errorf("%s: extends nested too deeply", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	base, _ := m["extends"].(string)
	if base == "" {
		return m, nil
	}
	parent, err := loadMerged(filepath.Join(filepath.Dir(path), base), depth+1)
	if err != nil {
		return nil, err
	}
	return merge(parent, m), nil
}

func merge(dst, src map[string]any) map[string]any {
	for k, v := range src {
		sm, ok1 := v.(map[string]any)
		dm, ok2 := dst[k].(map[string]any)
		if ok1 && ok2 {
			dst[k] = merge(dm, sm)
			continue
		}
		dst[k] = v
	}
	return dst
}

func expandAll(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = expandAll(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = expandAll(x)
		}
		return t
	case string:
		return os.Expand(t, lookup)
	default:
		return v
	}
}

func lookup(name string) string {
	key, def, hasDefault := strings.Cut(name, ":-")
	if v, ok := os.LookupEnv(key); ok && (v != "" || !hasDefault) {
		return v
	}
	return def
}

func (c *Config) validate() error {
	switch {
	case c.Environment == "":
		return fmt.Errorf("environment is required")
	case !strings.HasPrefix(c.Issuer, "http"):
		return fmt.Errorf("issuer must be an absolute URL, got %q", c.Issuer)
	case c.Orgs.Operators == "" || c.Orgs.Readers == "":
		return fmt.Errorf("orgs.operators and orgs.readers are required")
	}
	seen := map[string]bool{}
	for i, a := range c.Apps {
		if a.Project == "" {
			c.Apps[i].Project = a.Name
		}
		if a.Key == "" || a.Name == "" || len(a.RedirectURIs) == 0 {
			return fmt.Errorf("app %q: key, name and redirectUris are required", a.Key)
		}
		if seen[a.Key] {
			return fmt.Errorf("app %q defined twice", a.Key)
		}
		seen[a.Key] = true
	}
	return nil
}
