//go:build !envtest

package gameaction

import (
	"strings"
	"testing"
)

func FuzzResolve(f *testing.F) {
	// Seed corpus
	f.Add("hello", "string", "default", "hello")
	f.Add("test", "string", "", "")
	f.Add("123", "int", "0", "123")
	f.Add("true", "bool", "false", "true")
	f.Add("hello\nworld", "string", "", "")
	f.Add("\x00", "string", "", "")
	f.Add(strings.Repeat("a", 600), "string", "", "")
	f.Add("enumVal", "enum", "default", "enumVal")

	f.Fuzz(func(t *testing.T, val string, paramType string, defaultVal string, _ string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %q (type: %q, default: %q): %v", val, paramType, defaultVal, r)
			}
		}()

		decls := []Param{
			{
				Name:    "param1",
				Type:    paramType,
				Default: defaultVal,
				Enum:    []string{"enumVal", "otherVal"},
			},
		}

		got := map[string]string{
			"param1": val,
		}

		res, err := Resolve(decls, got)
		if err == nil {
			// Invariant check: Output should not contain shell metacharacters or RCON delimiters
			if strings.ToLower(strings.TrimSpace(paramType)) != "int" &&
				strings.ToLower(strings.TrimSpace(paramType)) != "bool" &&
				strings.ToLower(strings.TrimSpace(paramType)) != "enum" {
				if strings.ContainsAny(res["param1"], "\x00;&|$`\\\"'") {
					t.Errorf("Resolve allowed shell/rcon metacharacters in output: %q", res["param1"])
				}
			}
		}
	})
}
