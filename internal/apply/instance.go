package apply

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/kite-plus/identity/internal/config"
)

func (a *Applier) instance(ctx context.Context) error {
	var inst obj
	if err := a.call(ctx, http.MethodGet, "/admin/v1/instances/me", "", nil, &inst); err != nil {
		return err
	}
	a.instanceID = str(inst, "instance", "id")
	if a.state.InstanceID != a.instanceID {
		if a.state.InstanceID != "" {
			a.note("state belongs to instance %s, starting over for %s", a.state.InstanceID, a.instanceID)
		}
		a.state.reset(a.instanceID)
	}
	fmt.Fprintf(a.log, "  instance %s (%s)\n", a.instanceID, str(inst, "instance", "version"))

	c := a.cfg.Instance
	if c.DefaultLanguage != "" {
		var cur obj
		if err := a.call(ctx, http.MethodGet, "/admin/v1/languages/default", "", nil, &cur); err != nil {
			return err
		}
		if str(cur, "language") != c.DefaultLanguage {
			if err := a.call(ctx, http.MethodPut, "/admin/v1/languages/default/"+c.DefaultLanguage, "", obj{}, nil); err != nil {
				return err
			}
			a.changed("default language %s", c.DefaultLanguage)
		}
	}
	if len(c.AllowedLanguages) > 0 {
		var cur obj
		if err := a.call(ctx, http.MethodGet, "/admin/v1/restrictions", "", nil, &cur); err != nil {
			return err
		}
		if canon(strs(cur["allowedLanguages"])) != canon(c.AllowedLanguages) {
			in := obj{"allowedLanguages": obj{"list": c.AllowedLanguages}}
			if err := a.call(ctx, http.MethodPut, "/admin/v1/restrictions", "", in, nil); err != nil {
				return err
			}
			a.changed("allowed languages %v", c.AllowedLanguages)
		}
	}
	return nil
}

func (a *Applier) orgs(ctx context.Context) error {
	var err error
	if a.operatorsID, err = a.findOrg(ctx, a.cfg.Orgs.Operators); err != nil {
		return err
	}
	if a.operatorsID == "" {
		return fmt.Errorf("operators org %q not found; it is created by ZITADEL setup (FirstInstance.Org.Name)", a.cfg.Orgs.Operators)
	}
	if a.readersID, err = a.findOrg(ctx, a.cfg.Orgs.Readers); err != nil {
		return err
	}
	if a.readersID == "" {
		var out obj
		if err := a.call(ctx, http.MethodPost, "/v2/organizations", "", obj{"name": a.cfg.Orgs.Readers}, &out); err != nil {
			return err
		}
		a.readersID = str(out, "organizationId")
		a.changed("organization %q created", a.cfg.Orgs.Readers)
	}
	var def obj
	if err := a.call(ctx, http.MethodGet, "/admin/v1/orgs/default", "", nil, &def); err != nil {
		return err
	}
	if str(def, "org", "id") != a.readersID {
		if err := a.call(ctx, http.MethodPut, "/admin/v1/orgs/default/"+a.readersID, "", obj{}, nil); err != nil {
			return err
		}
		a.changed("default organization %q", a.cfg.Orgs.Readers)
	}
	return nil
}

func (a *Applier) findOrg(ctx context.Context, name string) (string, error) {
	var out obj
	in := obj{"queries": []obj{{"nameQuery": obj{"name": name, "method": textEquals}}}}
	if err := a.call(ctx, http.MethodPost, "/v2/organizations/_search", "", in, &out); err != nil {
		return "", err
	}
	for _, o := range list(out, "result") {
		if str(o, "name") == name {
			return str(o, "id"), nil
		}
	}
	return "", nil
}

// loginPolicyFields maps the config onto the fields shared by the instance
// and org login policy APIs.
func loginPolicyFields(p config.LoginPolicy) obj {
	pwless := "PASSWORDLESS_TYPE_NOT_ALLOWED"
	if p.Passkeys {
		pwless = "PASSWORDLESS_TYPE_ALLOWED"
	}
	return obj{
		"allowUsernamePassword":      p.AllowUsernamePassword,
		"allowRegister":              p.AllowRegister,
		"allowExternalIdp":           p.AllowExternalIdP,
		"forceMfa":                   p.ForceMFA,
		"passwordlessType":           pwless,
		"hidePasswordReset":          p.HidePasswordReset,
		"ignoreUnknownUsernames":     p.IgnoreUnknownUsernames,
		"allowDomainDiscovery":       p.AllowDomainDiscovery,
		"disableLoginWithEmail":      p.DisableLoginWithEmail,
		"disableLoginWithPhone":      p.DisableLoginWithPhone,
		"passwordCheckLifetime":      seconds(p.PasswordCheckLifetime.Std()),
		"externalLoginCheckLifetime": seconds(p.ExternalLoginCheckLifetime.Std()),
		"mfaInitSkipLifetime":        seconds(p.MFAInitSkipLifetime.Std()),
		"secondFactorCheckLifetime":  seconds(p.SecondFactorCheckLifetime.Std()),
		"multiFactorCheckLifetime":   seconds(p.MultiFactorCheckLifetime.Std()),
	}
}

func secondFactor(s string) string { return "SECOND_FACTOR_TYPE_" + strings.ToUpper(s) }
func multiFactor(s string) string  { return "MULTI_FACTOR_TYPE_" + strings.ToUpper(s) }

func mapStrings(in []string, f func(string) string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = f(s)
	}
	return out
}

func (a *Applier) loginPolicy(ctx context.Context) error {
	want := loginPolicyFields(a.cfg.LoginPolicy)
	var cur obj
	if err := a.call(ctx, http.MethodGet, "/admin/v1/policies/login", "", nil, &cur); err != nil {
		return err
	}
	pol := sub(cur, "policy")
	if d := diff(want, pol); len(d) > 0 {
		if err := a.call(ctx, http.MethodPut, "/admin/v1/policies/login", "", want, nil); err != nil {
			return err
		}
		a.changed("default login policy %v", d)
	}
	base := "/admin/v1/policies/login"
	if err := a.syncFactors(ctx, base+"/second_factors", "", strs(pol["secondFactors"]),
		mapStrings(a.cfg.LoginPolicy.SecondFactors, secondFactor), "second factor"); err != nil {
		return err
	}
	return a.syncFactors(ctx, base+"/multi_factors", "", strs(pol["multiFactors"]),
		mapStrings(a.cfg.LoginPolicy.MultiFactors, multiFactor), "multi factor")
}

func (a *Applier) syncFactors(ctx context.Context, path, orgID string, have, want []string, what string) error {
	for _, f := range want {
		if !slices.Contains(have, f) {
			if err := a.call(ctx, http.MethodPost, path, orgID, obj{"type": f}, nil); err != nil {
				return err
			}
			a.changed("%s %s added", what, f)
		}
	}
	for _, f := range have {
		if !slices.Contains(want, f) {
			if err := a.call(ctx, http.MethodDelete, path+"/"+f, orgID, nil, nil); err != nil {
				return err
			}
			a.changed("%s %s removed", what, f)
		}
	}
	return nil
}

// operatorLoginPolicy gives the operators org its own policy so admins can
// be forced to use MFA (R9) without forcing it on readers.
func (a *Applier) operatorLoginPolicy(ctx context.Context) error {
	p := a.cfg.LoginPolicy
	p.AllowRegister = false
	p.AllowExternalIdP = false
	p.ForceMFA = a.cfg.OperatorPolicy.ForceMFA
	if a.cfg.OperatorPolicy.PasswordCheckLifetime > 0 {
		p.PasswordCheckLifetime = a.cfg.OperatorPolicy.PasswordCheckLifetime
	}
	if a.cfg.OperatorPolicy.MultiFactorCheckLifetime > 0 {
		p.MultiFactorCheckLifetime = a.cfg.OperatorPolicy.MultiFactorCheckLifetime
	}
	want := loginPolicyFields(p)
	var cur obj
	if err := a.call(ctx, http.MethodGet, "/management/v1/policies/login", a.operatorsID, nil, &cur); err != nil {
		return err
	}
	pol := sub(cur, "policy")
	sf := mapStrings(p.SecondFactors, secondFactor)
	mf := mapStrings(p.MultiFactors, multiFactor)
	if pol["isDefault"] == true {
		in := maps.Clone(want)
		in["secondFactors"] = sf
		in["multiFactors"] = mf
		if err := a.call(ctx, http.MethodPost, "/management/v1/policies/login", a.operatorsID, in, nil); err != nil {
			return err
		}
		a.changed("operators login policy created")
		return nil
	}
	if d := diff(want, pol); len(d) > 0 {
		if err := a.call(ctx, http.MethodPut, "/management/v1/policies/login", a.operatorsID, want, nil); err != nil {
			return err
		}
		a.changed("operators login policy %v", d)
	}
	base := "/management/v1/policies/login"
	if err := a.syncFactors(ctx, base+"/second_factors", a.operatorsID, strs(pol["secondFactors"]), sf, "operators second factor"); err != nil {
		return err
	}
	return a.syncFactors(ctx, base+"/multi_factors", a.operatorsID, strs(pol["multiFactors"]), mf, "operators multi factor")
}

func (a *Applier) passwordPolicies(ctx context.Context) error {
	l := a.cfg.Lockout
	if err := a.syncPolicy(ctx, "/admin/v1/policies/lockout", "/admin/v1/policies/password/lockout", obj{
		"maxPasswordAttempts": u64(l.MaxPasswordAttempts),
		"maxOtpAttempts":      u64(l.MaxOTPAttempts),
	}, "lockout policy"); err != nil {
		return err
	}
	p := a.cfg.PasswordComplexity
	return a.syncPolicy(ctx, "/admin/v1/policies/password/complexity", "/admin/v1/policies/password/complexity", obj{
		"minLength":    u64(p.MinLength),
		"hasUppercase": p.HasUppercase,
		"hasLowercase": p.HasLowercase,
		"hasNumber":    p.HasNumber,
		"hasSymbol":    p.HasSymbol,
	}, "password complexity policy")
}

func (a *Applier) privacyPolicy(ctx context.Context) error {
	p := a.cfg.Privacy
	return a.syncPolicy(ctx, "/admin/v1/policies/privacy", "/admin/v1/policies/privacy", obj{
		"tosLink":      p.TOSLink,
		"privacyLink":  p.PrivacyLink,
		"helpLink":     p.HelpLink,
		"supportEmail": p.SupportEmail,
		"docsLink":     p.DocsLink,
	}, "privacy policy")
}

// syncPolicy reads {policy: ...} from getPath and PUTs want to putPath when
// any field differs.
func (a *Applier) syncPolicy(ctx context.Context, getPath, putPath string, want obj, what string) error {
	var cur obj
	if err := a.call(ctx, http.MethodGet, getPath, "", nil, &cur); err != nil {
		return err
	}
	if d := diff(want, sub(cur, "policy")); len(d) > 0 {
		if err := a.call(ctx, http.MethodPut, putPath, "", want, nil); err != nil {
			return err
		}
		a.changed("%s %v", what, d)
	}
	return nil
}
