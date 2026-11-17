package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func versionString() string {
	c := commit
	if c == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					c = s.Value[:7]
				}
			}
		}
	}

	s := version
	if c != "" {
		s += " (" + c + ")"
	}
	if date != "" {
		s += " built " + date
	}
	return fmt.Sprintf("%s %s", s, runtime.Version())
}
