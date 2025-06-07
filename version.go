package caddydns01proxy

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

const caddydns01proxyPackagePath = "github.com/liujed/caddy-dns01proxy"

// Returns the version and go-mod hash. For example, "v0.0.0 (h1:abcd1234=)".
func Version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	for _, dep := range buildInfo.Deps {
		if dep.Path == caddydns01proxyPackagePath {
			for dep.Replace != nil {
				dep = dep.Replace
			}
			buf := strings.Builder{}
			buf.WriteString(dep.Version)
			if dep.Sum != "" {
				buf.WriteString(" (")
				buf.WriteString(dep.Sum)
				buf.WriteString(")")
			}
			return buf.String()
		}
	}

	return "unknown"
}

// Returns the release string, including the application name, version, go-mod
// hash, OS, and architecture. For example, "dns01proxy v0.0.0 (h1:abcd1234=)
// linux/amd64".
func Release() string {
	return fmt.Sprintf(
		"dns01proxy %s %s/%s",
		Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
