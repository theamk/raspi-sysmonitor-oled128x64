package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

func versionString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown (no build info)"
	}
	s := info.Main.Path
	s += fmt.Sprintf("\ngo: %s %s/%s", info.GoVersion, runtime.GOOS, runtime.GOARCH)
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			s += "\nrevision: " + kv.Value
		case "vcs.time":
			s += "\nbuilt from commit time: " + kv.Value
		case "vcs.modified":
			s += "\nmodified: " + kv.Value
		}
	}
	return s
}
