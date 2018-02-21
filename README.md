# envflag [![Travis-CI](https://travis-ci.org/dcowgill/envflag.svg)](https://travis-ci.org/dcowgill/envflag) [![GoDoc](https://godoc.org/github.com/dcowgill/envflag?status.svg)](http://godoc.org/github.com/dcowgill/envflag) [![Report card](https://goreportcard.com/badge/github.com/dcowgill/envflag)](https://goreportcard.com/report/github.com/dcowgill/envflag)

Minimalist approach to wiring up the standard flag package to the environment.

## Usage

```go
func main() {
    listenAddr := flag.String("addr", ":8080", "server listen address")
    flag.Parse()

    envflag.SetPrefix("myapp")
    envflag.Parse()

    fmt.Printf("listenAddr is %q\n", *listenAddr)
}
```

## Rationale

This package connects the [standard flag package](https://godoc.org/flag) to the
environment with a minimum of ceremony.

[12-factor](https://12factor.net/config) apps store their configuration in the
environment. It's also useful, however, to accept command-line flags: they make
a program easier to use, and serve as documentation.

## Example

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dcowgill/envflag"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	listenAddr := fs.String("addr", ":8888", "server listen address")
	timeout := fs.Int("timeout", 10, "client timeout in seconds")
	fs.Parse(nil) // simulate no command-line args

	// Normally this would happen outside the program.
	os.Setenv("MYAPP_LISTEN_ADDR", ":9999")
	os.Setenv("MYADD_TIMEOUT", "42")

	vs := envflag.NewVarSet(fs)
	vs.SetPrefix("myapp")
	vs.RenameFlag("addr", "listen-addr")
	vs.Parse()

	fmt.Println(*listenAddr) // prints ":9999"
	fmt.Println(*timeout)    // prints "42"
}
```
