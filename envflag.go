/*

Package envflag parses command-line flags defined by package flag from the environment.

Usage:

	package main

	import (
		"flag"
		"fmt"

		"github.com/dcowgill/envflag"
	)

	func main() {
		listenAddr := flag.String("listen-addr", ":8080", "server listen address")
		flag.Parse()
		envflag.SetPrefix("myapp")
		envflag.Parse()
		fmt.Printf("listenAddr is %q\n", *listenAddr)
	}

An environment variable will not override a flag, but will override its default value:

	$ go run example.go
	listenAddr is ":8080"
	$ MYAPP_LISTEN_ADDR=:7070 go run example.go
	listenAddr is ":7070"
	$ MYAPP_LISTEN_ADDR=:7070 go run example.go -listen-addr=:9090
	listenAddr is ":9090"

Flag names are automatically converted to environment variable keys according to
the following rules:

	- Non-ASCII runes are omitted.
	- Uppercase letters, digits, and underscores are preserved.
	- Lowercase letters are changed to uppercase.
	- Hyphens are changed to underscores.
	- All other runes are omitted.
	- Prepend an underscore if a variable name would otherwise begin with a digit.

*/
package envflag

import (
	"strings"
	"flag"
	"fmt"
	"os"
)

// A VarSet wraps a flag.FlagSet. The zero value of a VarSet should not be used.
//
// A program that uses the top-level functions in package flag should also use
// the top-level functions in this package. Otherwise, it must wrap each
// flag.FlagSet with a VarSet to connect it to the environment.
type VarSet struct {
	// LookupEnv is an optional replacement for os.LookupEnv in this VarSet.
	LookupEnv func(key string) (string, bool)

	fs            *flag.FlagSet
	prefix        string
	renames       map[string]string
}

// NewVarSet creates a new VarSet with the specified flag set and error handling property.
func NewVarSet(fs *flag.FlagSet) *VarSet {
	return &VarSet{fs: fs}
}

// SetPrefix specifies a string to prepend to all environment variable keys. An
// underscore is automatically inserted between the prefix and variable key.
func (vs *VarSet) SetPrefix(prefix string) {
	vs.prefix = prefix
}

// RenameFlag modifies a flag name before it is converted to an environment key.
// The new name will be transformed by the same process as any other flag name.
func (vs *VarSet) RenameFlag(old, new string) {
	if vs.renames == nil {
		vs.renames = make(map[string]string)
	}
	vs.renames[old] = new
}

// Parse sets the value of flags that were not provided on the command-line but
// are set in the environment.
func (vs *VarSet) Parse() error {
	flags := make(map[string]*flag.Flag)
	vs.fs.VisitAll(func(f *flag.Flag) {
		flags[f.Name] = f // collect all flags in a map
	})
	vs.fs.Visit(func(f *flag.Flag) {
		delete(flags, f.Name) // remove flags that were set
	})
	for _, f := range flags {
		if err := vs.parseOne(f.Name, f.Value); err != nil {
			switch vs.fs.ErrorHandling() {
			case flag.ContinueOnError:
				return err
			case flag.ExitOnError:
				os.Exit(2)
			case flag.PanicOnError:
				panic(err)
			}
		}
	}
	return nil
}

// Retrieves the value of the environment variable associated with the specified
// flag and, if the variable is set, stores its current value in dst.
func (vs *VarSet) parseOne(flagName string, dst flag.Value) error {
	key := rewrite(vs.prefix, vs.renamed(flagName))
	if value, found := vs.lookupEnv(key); found {
		if err := dst.Set(value); err != nil {
			return vs.failf("invalid value %q for environment variable %q: %v", value, key, err)
		}
	}
	return nil
}

// If the flag was renamed by vs.Rename, reports its new name.
func (vs *VarSet) renamed(flagName string) string {
	if s := vs.renames[flagName]; s != "" {
		return s
	}
	return flagName
}

// Calls vs.LookupEnv, or os.LookupEnv if vs.LookupEnv is nil.
func (vs *VarSet) lookupEnv(key string) (string, bool) {
	if vs.LookupEnv != nil {
		return vs.LookupEnv(key)
	}
	return os.LookupEnv(key)
}

func (vs *VarSet) failf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	fmt.Fprintln(vs.fs.Output(), err)
	vs.fs.Usage()
	return err
}

// CommandLine wraps flag.CommandLine, the default set of command-line flags.
// The top-level functions, such as SetPrefix and Parse, are wrappers for the
// methods of CommandLine.
var CommandLine = NewVarSet(flag.CommandLine)

// SetPrefix specifies a string to prepend to all environment variable keys. An
// underscore is automatically inserted between the prefix and variable key.
func SetPrefix(prefix string) {
	CommandLine.SetPrefix(prefix)
}

// RenameFlag modifies a flag name before it is converted to an environment key.
// The new name will be transformed by the same process as any other flag name.
func RenameFlag(old, new string) {
	CommandLine.RenameFlag(old, new)
}

// Parse sets the value of flags that were not provided on the command-line but
// are set in the environment.
func Parse() {
	_ = CommandLine.Parse() // default behavior is ExitOnError
}

// Transforms a flag name, plus an optional prefix, into an environment key.
func rewrite(prefix, name string) string {
	b := strings.Builder{}
	b.Grow(len(prefix) + len(name) + 1)
	if prefix != "" {
		rewriteInto(&b, prefix)
		b.WriteByte('_')
	}
	rewriteInto(&b, name)
	return b.String()
}

func rewriteInto(b *strings.Builder, s string) {
	for _, ch := range s {
		switch {
		case ch >= 'A' && ch <= 'Z':
		case ch >= 'a' && ch <= 'z':
			ch = 'A' + ch - 'a'
		case ch >= '0' && ch <= '9':
			if b.Len() == 0 {
				b.WriteByte('_') // cannot begin with a digit
			}
		case ch == '_':
		case ch == '-':
			ch = '_'
		default:
			continue // character not permitted
		}
		b.WriteByte(byte(ch))
	}
}
