package envflag

import (
	"flag"
	"fmt"
	"strconv"
	"testing"
)

// Verifies that command-line flags take precedence over the environment.
func TestPrecedence(t *testing.T) {
	const (
		defaultValue     = 123
		flagValue        = 456
		environmentValue = 789
	)
	// Define two helper functions to remove repetitive boilerplate.
	parseFlags := func(set bool) (*flag.FlagSet, *int) {
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		value := fs.Int("foo", defaultValue, "")
		var arguments []string
		if set {
			arguments = []string{fmt.Sprintf("-foo=%d", flagValue)}
		}
		must(fs.Parse(arguments))
		return fs, value
	}
	parseEnv := func(fs *flag.FlagSet, value string, set bool) {
		vs := NewVarSet(fs)
		vs.LookupEnv = func(string) (string, bool) { return value, set }
		must(vs.Parse())
	}
	expectEq := func(value *int, expected int) {
		if *value != expected {
			t.Errorf("flag -foo is %d, want %d", *value, expected)
		}
	}
	// Test various scenarios.
	t.Run("neither flag nor environment set", func(t *testing.T) {
		fs, value := parseFlags(false)
		parseEnv(fs, "", false)
		expectEq(value, defaultValue)
	})
	t.Run("only flag set", func(t *testing.T) {
		fs, value := parseFlags(true)
		parseEnv(fs, "", false)
		expectEq(value, flagValue)
	})
	t.Run("only environment set", func(t *testing.T) {
		fs, value := parseFlags(false)
		parseEnv(fs, strconv.Itoa(environmentValue), true)
		expectEq(value, environmentValue)
	})
	t.Run("both flag and environment set", func(t *testing.T) {
		fs, value := parseFlags(true)
		parseEnv(fs, strconv.Itoa(environmentValue), true)
		expectEq(value, flagValue)
	})
}

// Verifies that an empty environment variable overrides a flag default.
func TestEmptyButSetEnvironment(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	value := fs.String("foo", "the default", "")
	must(fs.Parse(nil))
	vs := NewVarSet(fs)
	vs.LookupEnv = func(string) (string, bool) { return "", true }
	must(vs.Parse())
	if *value != "" {
		t.Errorf("flag -foo is %q, want empty string", *value)
	}
}

// Tests that renaming a flag works.
func TestRenameFlag(t *testing.T) {
	const environmentValue = "hello, world!"
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	value := fs.String("original", "", "")
	must(fs.Parse(nil))
	vs := NewVarSet(fs)
	vs.SetPrefix("testapp")
	vs.RenameFlag("original", "new-and-improved")
	vs.LookupEnv = func(key string) (string, bool) {
		if key == "TESTAPP_NEW_AND_IMPROVED" {
			return environmentValue, true
		}
		return "", false
	}
	must(vs.Parse())
	if *value != environmentValue {
		t.Errorf("flag -original is %q, want %q", *value, environmentValue)
	}
}

// Verifies the rules of flag-to-environment name rewriting.
func TestRewrite(t *testing.T) {
	var tests = []struct {
		desc, prefix, name, key string
	}{
		{"empty", "", "", ""},
		{"simple", "", "foo", "FOO"},
		{"simple with prefix", "foo", "bar", "FOO_BAR"},
		{"name starts with digit", "", "0a", "_0A"},
		{"prefix starts with digit", "0a", "foo", "_0A_FOO"},
		{"prefix and name start with digit", "0a", "0a", "_0A_0A"},
		{"mixed case", "aBcD", "EfGh", "ABCD_EFGH"},
		{"hyphens", "", "foo-bar-qux-", "FOO_BAR_QUX_"},
		{"prefix hyphens", "-foo-bar-", "-baz", "_FOO_BAR___BAZ"},
		{"illegal runes", "", "Hello, 世界", "HELLO"},
		{"illegal runes with prefix", "Hello, ", "世界", "HELLO_"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if key := rewrite(tt.prefix, tt.name); key != tt.key {
				t.Errorf("rewrite(%q, %q) returned %q, want %q", tt.prefix, tt.name, key, tt.key)
			}
		})
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
