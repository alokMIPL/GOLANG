package main

import (
	"flag"
	"io"
)

type options struct {
	showVersion bool
	migrateOnly bool
	healthcheck bool
}

func parseFlags(args []string, out io.Writer) (options, error) {
	var o options

	fs := flag.NewFlagSet("taskapi", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&o.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&o.migrateOnly, "migrate-only", false, "apply database migrations and exit")
	fs.BoolVar(&o.healthcheck, "healthcheck", false,
		"probe the local /healthz endpoint and exit 0 if healthy (for container HEALTHCHECK)")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return o, nil
}
