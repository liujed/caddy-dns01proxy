package caddydns01proxy

import (
	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// A Caddy `http.handlers` module that implements the dns01proxy API.
type Handler struct {
}

var _ caddy.Module = (*Handler)(nil)
var _ caddy.Provisioner = (*Handler)(nil)

func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.dns01proxy",
		New: func() caddy.Module {
			return new(Handler)
		},
	}
}

func (h *Handler) Provision(ctx caddy.Context) error {
	return nil
}
