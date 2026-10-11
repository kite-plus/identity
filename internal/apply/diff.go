package apply

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type obj = map[string]any

// diff lists the keys of want whose value differs from have. The gateway
// omits zero values and encodes 64-bit integers as strings, so absent,
// false, "", "0" and empty lists all count as the same zero value, and
// lists compare as sets.
func diff(want, have obj) []string {
	var out []string
	for k, w := range want {
		if canon(w) != canon(have[k]) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

func canon(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case bool:
		if !t {
			return ""
		}
		return "true"
	case string:
		if t == "0" || t == "0s" {
			return ""
		}
		return t
	case float64:
		if t == 0 {
			return ""
		}
		return fmt.Sprint(t)
	case int, int64, uint64, uint32:
		s := fmt.Sprint(t)
		if s == "0" {
			return ""
		}
		return s
	case []string:
		s := slices.Clone(t)
		slices.Sort(s)
		return strings.Join(s, "\x00")
	case []any:
		s := make([]string, 0, len(t))
		for _, x := range t {
			s = append(s, canon(x))
		}
		slices.Sort(s)
		return strings.Join(s, "\x00")
	case obj:
		keys := make([]string, 0, len(t))
		for k, x := range t {
			if canon(x) != "" {
				keys = append(keys, k+"="+canon(x))
			}
		}
		slices.Sort(keys)
		return "{" + strings.Join(keys, ",") + "}"
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// seconds formats a duration the way the gateway returns
// google.protobuf.Duration values.
func seconds(d time.Duration) string {
	return fmt.Sprintf("%ds", int64(d/time.Second))
}

func u64(v uint64) string { return fmt.Sprint(v) }

func strs(v any) []string {
	l, _ := v.([]any)
	out := make([]string, 0, len(l))
	for _, x := range l {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func str(m obj, path ...string) string {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(obj)
		if !ok {
			return ""
		}
		cur = mm[p]
	}
	s, _ := cur.(string)
	return s
}

func sub(m obj, key string) obj {
	v, _ := m[key].(obj)
	if v == nil {
		return obj{}
	}
	return v
}

func list(m obj, key string) []obj {
	l, _ := m[key].([]any)
	out := make([]obj, 0, len(l))
	for _, x := range l {
		if o, ok := x.(obj); ok {
			out = append(out, o)
		}
	}
	return out
}
