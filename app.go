package caddydns01proxy

import (
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(App{})
}

// A Caddy application module that implements the dns01proxy server.
type App struct {
	Handler

	// The sockets on which to listen.
	Listen []string `json:"listen"`

	// Configures the set of trusted proxies.
	TrustedProxiesRaw json.RawMessage `json:"trusted_proxies,omitempty" caddy:"namespace=http.ip_sources inline_key=source"`
}

var _ caddy.Module = (*App)(nil)

func (App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "dns01proxy",
		New: func() caddy.Module {
			return new(App)
		},
	}
}
