package caddydns01proxy

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
)

type DNSConfig struct {
	// The DNS provider for publishing DNS-01 responses.
	ProviderRaw json.RawMessage `json:"provider" caddy:"namespace=dns.providers inline_key=name"`

	Provider certmagic.DNSProvider `json:"-"`

	// The TTL to use in DNS TXT records when answering challenges. Optional. Not
	// usually needed.
	TTL *caddy.Duration `json:"ttl,omitempty"`

	// Custom DNS resolvers to prefer over system or built-in defaults. Set this
	// to a public resolver if you are using split-horizon DNS. Remember to also
	// configure your ACME clients' resolvers, since both the ACME client and
	// dns01proxy need to find your domain's SOA record.
	Resolvers []string `json:"resolvers,omitempty"`

	// How long to wait before starting to check whether the DNS record has
	// propagated. Optional. Only used when the dns01proxy app obtains its own
	// certificate; ACME clients of the proxy run their own checks. Useful for
	// DNS providers whose authoritative servers pick up a new record at
	// different speeds, where the propagation check can succeed against one
	// server while the CA still sees NXDOMAIN at another.
	PropagationDelay caddy.Duration `json:"propagation_delay,omitempty"`

	// Maximum time to wait for the DNS record to propagate. Optional. Only
	// used when the dns01proxy app obtains its own certificate. Set to -1 to
	// disable propagation checks.
	PropagationTimeout caddy.Duration `json:"propagation_timeout,omitempty"`
}

var _ caddy.Provisioner = (*Handler)(nil)

func (d *DNSConfig) Provision(ctx caddy.Context) error {
	if len(d.ProviderRaw) == 0 {
		return fmt.Errorf("must configure a DNS provider")
	}

	module, err := ctx.LoadModule(d, "ProviderRaw")
	if err != nil {
		return fmt.Errorf("unable to load DNS provider: %w", err)
	}
	d.Provider = module.(certmagic.DNSProvider)

	return nil
}
