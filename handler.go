package caddydns01proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/caddyauth"
	"github.com/caddyserver/certmagic"
	"github.com/libdns/libdns"
	"github.com/liujed/caddy-dns01proxy/jsonutil"
	"github.com/liujed/goutil/optionals"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Handler{})
	httpcaddyfile.RegisterHandlerDirective("dns01proxy", parseHandler)
	httpcaddyfile.RegisterDirectiveOrder(
		"dns01proxy",
		httpcaddyfile.After,
		"acme_server",
	)
}

// Implements an API for proxying ACME DNS-01 challenges.
//
// This is a Caddy `http.handlers` module.
type Handler struct {
	DNS DNSConfig `json:"dns"`

	// Configures HTTP basic authentication and the domains for which each user
	// can get TLS certificates.
	//
	// (During provisioning, this is used to fill in [Authentication] and
	// [ClientRegistry].)
	AccountsRaw []RawAccount `json:"accounts"`

	// Specifies how clients should be authenticated. If absent, then clients must
	// be authenticated by an `http.handlers.authentication` instance earlier in
	// the handler chain. Derived from [AccountsRaw].
	Authentication optionals.Optional[*caddyauth.Authentication] `json:"-"`

	// Identifies the domains at which each client is allowed to answer DNS-01
	// challenges. Derived from [AccountsRaw].
	ClientRegistry ClientRegistry `json:"-"`

	logger *zap.Logger
}

var _ caddy.Module = (*Handler)(nil)
var _ caddy.Provisioner = (*Handler)(nil)
var _ caddyhttp.MiddlewareHandler = (*Handler)(nil)
var _ caddyfile.Unmarshaler = (*Handler)(nil)

type RawAccount struct {
	ClientPolicy

	// The user's password, hashed using `caddy hash-password`. Optional. If
	// omitted, then clients must be authenticated by an
	// `http.handlers.authentication` instance earlier in the handler chain.
	Password *string `json:"password,omitempty"`
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
	h.logger = ctx.Logger()

	// Provision DNS.
	err := h.DNS.Provision(ctx)
	if err != nil {
		return err
	}

	// Provision Authentication from AccountsRaw.
	accountList := []caddyauth.Account{}
	for _, rawAccount := range h.AccountsRaw {
		if rawAccount.Password != nil {
			accountList = append(accountList, caddyauth.Account{
				Username: rawAccount.UserID,
				Password: *rawAccount.Password,
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
	nextHandler caddyhttp.Handler,
) error {
	var mode handlerMode
	switch req.URL.Path {
	case "/present":
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return nil
		}
		mode = hmPresent

	case "/cleanup":
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return nil
		}
		mode = hmCleanup

	default:
		return nextHandler.ServeHTTP(w, req)
	}

	handlerImpl := jsonutil.WrapHandler(h.handleDNSRequest(mode))
	if auth, exists := h.Authentication.Get(); exists {
		return auth.ServeHTTP(w, req, handlerImpl)
	}
	return handlerImpl.ServeHTTP(w, req)
}

type handlerMode string

const (
	hmPresent handlerMode = "present"
	hmCleanup handlerMode = "cleanup"
)

func (h *Handler) handleDNSRequest(
	mode handlerMode,
) func(*http.Request, RequestBody) (int, optionals.Optional[ResponseBody], error) {
	return func(
		req *http.Request,
		reqBody RequestBody,
	) (httpStatus int, respBody optionals.Optional[ResponseBody], err error) {
		// Check that the user gave a valid request body.
		if !reqBody.IsValid() {
			return http.StatusBadRequest, optionals.None[ResponseBody](), nil
		}

		// Log the challenge domain that appears in the request.
		addLogField(req, zap.String("domain", reqBody.ChallengeFQDN))

		// Check that the user is authorized for the challenge domain in the
		// request.
		denyReasonOpt, err := h.ClientRegistry.AuthorizeUserChallengeDomain(
			req,
			reqBody.ChallengeFQDN,
		)
		if err != nil {
			return 0, optionals.None[ResponseBody](),
				fmt.Errorf("unable to authorize user for requested domain: %w", err)
		}
		if denyReason, denied := denyReasonOpt.Get(); denied {
			addLogField(req, zap.String(logAuthorizationFailure, string(denyReason)))
			return http.StatusForbidden, optionals.None[ResponseBody](), nil
		}

		// Figure out the challenge domain's DNS zone.
		zone, err := certmagic.FindZoneByFQDN(
			req.Context(),
			h.logger,
			reqBody.ChallengeFQDN,
			certmagic.RecursiveNameservers(h.DNS.Resolvers),
		)
		if err != nil {
			return 0, optionals.None[ResponseBody](),
				fmt.Errorf(
					"unable to find DNS zone for %q: %w",
					reqBody.ChallengeFQDN,
					err,
				)
		}

		// Build the DNS record to create/delete.
		ttl := time.Duration(0)
		if mode != hmCleanup && h.DNS.TTL != nil {
			ttl = time.Duration(*h.DNS.TTL)
		}
		records := []libdns.Record{
			libdns.TXT{
				Name: libdns.RelativeName(reqBody.ChallengeFQDN, zone),
				TTL:  ttl,
				Text: `"` + reqBody.Value + `"`,
			},
		}

		switch mode {
		case hmPresent:
			// Create the DNS record.
			_, err = h.DNS.Provider.AppendRecords(req.Context(), zone, records)
			if err != nil {
				return 0, optionals.None[ResponseBody](),
					fmt.Errorf("error creating DNS record: %w", err)
			}
			return http.StatusOK, optionals.Some(reqBody), nil

		case hmCleanup:
			// Delete the DNS record.
			_, err = h.DNS.Provider.DeleteRecords(req.Context(), zone, records)
			if err != nil {
				return 0, optionals.None[ResponseBody](),
					fmt.Errorf("error deleting DNS record: %w", err)
			}
			return http.StatusOK, optionals.Some(reqBody), nil
		}

		return 0, optionals.None[ResponseBody](),
			fmt.Errorf("unknown handler mode: %q", mode)
	}
}

// Parses a dns01proxy directive into a Handler instance.
//
// Syntax:
//
//	dns01proxy {
//		dns <provider_name> [<params...>]
//		dns_ttl <ttl>
//		resolvers <resolvers...>
//		user <userID> {
//			password <hashed_password>
//			allow_domains <domains...>
//			deny_domains <domains...>
//		}
//	}
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// Consume directive name.
	d.Next()

	// No inline arguments allowed.
	if d.NextArg() {
		return d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "dns":
			// Expect a provider name.
			if !d.NextArg() {
				return d.ArgErr()
			}
			provName := d.Val()
			unm, err := caddyfile.UnmarshalModule(d, "dns.providers."+provName)
			if err != nil {
				return err
			}
			h.DNS.ProviderRaw = caddyconfig.JSONModuleObject(
				unm,
				"name",
				provName,
				nil,
			)

		case "dns_ttl":
			var ttl string
			if !d.AllArgs(&ttl) {
				return d.ArgErr()
			}
			parsedTTL, err := caddy.ParseDuration(ttl)
			if err != nil {
				return err
			}
			caddyTTL := caddy.Duration(parsedTTL)
			h.DNS.TTL = &caddyTTL

		case "resolvers":
			h.DNS.Resolvers = d.RemainingArgs()
			if len(h.DNS.Resolvers) == 0 {
				return d.Errf("must specify at least one resolver address")
			}

		case "user":
			var userID string
			if !d.AllArgs(&userID) {
				return d.ArgErr()
			}

			account := RawAccount{
				ClientPolicy: ClientPolicy{
					UserID: userID,
				},
			}

			// Parse the client declaration.
			for nesting := d.Nesting(); d.NextBlock(nesting); {
				var curDomainsRaw *[]string

				fieldName := d.Val()
				switch fieldName {
				case "password":
					var password string
					if !d.AllArgs((&password)) {
						return d.ArgErr()
					}
					if account.Password != nil {
						return fmt.Errorf("cannot specify more than one password per user")
					}
					account.Password = &password
					continue

				case "allow_domains":
					curDomainsRaw = &account.AllowDomainsRaw

				case "deny_domains":
					curDomainsRaw = &account.DenyDomainsRaw

				default:
					return d.Errf("unrecognized user directive: %q", fieldName)
				}

				if *curDomainsRaw != nil {
					return d.Errf(
						"cannot specify more than one %q policy per user",
						fieldName,
					)
				}

				domainList := d.RemainingArgs()
				if len(domainList) == 0 {
					return d.Errf("must specify at least one domain")
				}

				*curDomainsRaw = domainList
			}

			// Register the account.
			h.AccountsRaw = append(h.AccountsRaw, account)

		default:
			return d.Errf("unrecognized dns01proxy handler directive: %q", d.Val())
		}
	}

	return nil
}

// Unmarshals tokens from h into a new Handler instance that is ready for
// provisioning.
func parseHandler(
	h httpcaddyfile.Helper,
) (caddyhttp.MiddlewareHandler, error) {
	var result Handler
	err := result.UnmarshalCaddyfile(h.Dispenser)

	if len(result.DNS.ProviderRaw) == 0 {
		// No locally configured DNS provider. Use the global option.
		val := h.Option("acme_dns")
		if val == nil {
			val = h.Option("dns")
			if val == nil {
				return nil, fmt.Errorf("must configure a DNS provider")
			}
		}
		result.DNS.ProviderRaw = caddyconfig.JSONModuleObject(
			val,
			"name",
			val.(caddy.Module).CaddyModule().ID.Name(),
			nil,
		)
	}

	return &result, err
}
