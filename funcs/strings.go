package funcs

import (
	"fmt"
	"strings"
)

// StringFuncMap return string func map.
func StringFuncMap() map[string]interface{} {
	return map[string]interface{}{
		"upper":       strings.ToUpper,
		"lower":       strings.ToLower,
		"string":      func(v interface{}) string { return fmt.Sprintf("%v", v) },
		"trim":        func(c, s string) string { return strings.Trim(s, c) },
		"trim_left":   func(c, s string) string { return strings.TrimLeft(s, c) },
		"trim_right":  func(c, s string) string { return strings.TrimRight(s, c) },
		"trim_prefix": func(c, s string) string { return strings.TrimPrefix(s, c) },
		"trim_suffix": func(c, s string) string { return strings.TrimSuffix(s, c) },
		"title":       strings.Title,
		"fields":      strings.Fields,
		"wc":          func(s string) int { return len(strings.Fields(s)) },
		"has_prefix":  func(c, s string) bool { return strings.HasPrefix(s, c) },
		"has_suffix":  func(c, s string) bool { return strings.HasSuffix(s, c) },
		"replace":     func(old, new string, n int, s string) string { return strings.Replace(s, old, new, n) },
		"replace_all": func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
		"count":       func(sub, s string) int { return strings.Count(s, sub) },
		"split":       func(sep, s string) []string { return strings.Split(s, sep) },
		"split_n":     func(sep string, n int, s string) []string { return strings.SplitN(s, sep, n) },
	}
}
