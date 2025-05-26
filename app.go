package caddydns01proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
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

	// The http module instance that implements this app.
	httpApp *caddyhttp.App `json:"-"`
}

var _ caddy.Module = (*App)(nil)
var _ caddy.Provisioner = (*App)(nil)
var _ caddy.App = (*App)(nil)

func (App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "dns01proxy",
		New: func() caddy.Module {
			return new(App)
		},
	}
}

func (app *App) Provision(ctx caddy.Context) error {
	module, err := ctx.LoadModuleByID(
		"http",
		caddyconfig.JSON(
			caddyhttp.App{
				Servers: map[string]*caddyhttp.Server{
					"dns01proxy": {
						Listen:            app.Listen,
						Routes:            app.makeRoutes(),
						TrustedProxiesRaw: app.TrustedProxiesRaw,

						// Turns on logging.
						Logs: &caddyhttp.ServerLogConfig{},
					},
				},
			},
			nil,
		),
	)
	if err != nil {
		return fmt.Errorf("unable to load http guest module: %w", err)
	}

	app.httpApp = module.(*caddyhttp.App)
	return nil
}

func (app *App) Start() error {
	return app.httpApp.Start()
}

func (app *App) Stop() error {
	return app.httpApp.Stop()
}

func (app *App) makeRoutes() caddyhttp.RouteList {
	return caddyhttp.RouteList{
		{
			HandlersRaw: []json.RawMessage{
				caddyconfig.JSONModuleObject(
					app.Handler,
					"handler",
					"dns01proxy",
					nil,
				),
				caddyconfig.JSONModuleObject(
					caddyhttp.StaticResponse{
						StatusCode: caddyhttp.WeakString(strconv.Itoa(
							http.StatusNotFound,
						)),
					},
					"handler",
					"static_response",
					nil,
				),
			},
		},
	}
}
