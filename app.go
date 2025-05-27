package caddydns01proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
)

func init() {
	caddy.RegisterModule(App{})
}

// A Caddy application module that implements the dns01proxy server.
type App struct {
	Handler

	// The server's hostnames. Used for obtaining TLS certificates.
	Hostnames []string `json:"hostnames"`

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

						// Turn off HTTP-to-HTTPS redirection. It masks insecure client
						// configurations.
						AutoHTTPS: &caddyhttp.AutoHTTPSConfig{
							DisableRedir: true,
						},

						// Turns on logging.
						Logs: &caddyhttp.ServerLogConfig{},

						// Turns on TLS.
						TLSConnPolicies: caddytls.ConnectionPolicies{
							&caddytls.ConnectionPolicy{},
						},
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
			MatcherSetsRaw: caddyhttp.RawMatcherSets{
				{
					"host": caddyconfig.JSON(
						app.Hostnames,
						nil,
					),
				},
			},
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

// Returns a TLS app configuration that uses the user-specified DNS provider for
// ACME challenges during TLS automation.
func (app *App) MakeTLSConfig() caddytls.TLS {
	return caddytls.TLS{
		Automation: &caddytls.AutomationConfig{
			Policies: []*caddytls.AutomationPolicy{
				{
					IssuersRaw: []json.RawMessage{
						caddyconfig.JSONModuleObject(
							caddytls.ACMEIssuer{
								Challenges: &caddytls.ChallengesConfig{
									DNS: &caddytls.DNSChallengeConfig{
										ProviderRaw: app.DNS.ProviderRaw,
										Resolvers:   app.DNS.Resolvers,
									},
								},
							},
							"module",
							"acme",
							nil,
						),
					},
				},
			},
		},
	}
}
