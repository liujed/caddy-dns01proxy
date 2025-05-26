package caddydns01proxy

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/caddyauth"
	"github.com/liujed/goutil/optionals"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// A Caddy `http.handlers` module that implements the dns01proxy API.
type Handler struct {
	DNS DNSConfig `json:"dns"`

	// During provisioning, this is used to fill in [Authentication] and
	// [ClientRegistry].
	AccountsRaw []RawAccount `json:"accounts"`

	// Specifies how clients should be authenticated. If absent, then clients must
	// be authenticated by an `http.handlers.authentication` instance earlier in
	// the handler chain. Derived from [AccountsRaw].
	Authentication optionals.Optional[*caddyauth.Authentication] `json:"-"`

	// Identifies the domains at which each client is allowed to answer DNS-01
	// challenges. Derived from [AccountsRaw].
	ClientRegistry ClientRegistry `json:"-"`
}

var _ caddy.Module = (*Handler)(nil)
var _ caddy.Provisioner = (*Handler)(nil)
var _ caddyhttp.MiddlewareHandler = (*Handler)(nil)

type RawAccount struct {
	ClientPolicy
	Password optionals.Optional[string] `json:"password"`
}

func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.dns01proxy",
		New: func() caddy.Module {
			return new(Handler)
		},
	}
}

func (h *Handler) Provision(ctx caddy.Context) error {
	// Provision DNS.
	err := h.DNS.Provision(ctx)
	if err != nil {
		return err
	}

	// Provision Authentication from AccountsRaw.
	accountList := []caddyauth.Account{}
	for _, rawAccount := range h.AccountsRaw {
		if password, exists := rawAccount.Password.Get(); exists {
			accountList = append(accountList, caddyauth.Account{
				Username: rawAccount.UserID,
				Password: password,
			})
		}
	}
	if len(accountList) > 0 {
		auth := &caddyauth.Authentication{
			ProvidersRaw: caddy.ModuleMap{
				"http_basic": caddyconfig.JSON(
					caddyauth.HTTPBasicAuth{
						AccountList: accountList,
					},
					nil,
				),
			},
		}
		err := auth.Provision(ctx)
		if err != nil {
			return fmt.Errorf("unable to provision authenticaiton: %w", err)
		}

		h.Authentication = optionals.Some(auth)
	}

	// Normally, we expect either all users or no users to have a password
	// configured. Warn if this is not the case.
	if len(accountList) > 0 && len(accountList) != len(h.AccountsRaw) {
		ctx.Logger().Warn("some users will always fail authentication because they do not have a password configured")
	}

	// Provision ClientRegistry from AccountsRaw.
	err = h.ClientRegistry.Provision(ctx, h.AccountsRaw)
	if err != nil {
		return fmt.Errorf("unable to provision client registry: %w", err)
	}

	// Allow AccountsRaw to be GC'd.
	h.AccountsRaw = nil

	return nil
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	req *http.Request,
	_ caddyhttp.Handler,
) error {
	w.WriteHeader(http.StatusNotFound)
	return nil
}
