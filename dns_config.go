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
	// to a public resolver if you are using split-horizon DNS.
	Resolvers []string `json:"resolvers,omitempty"`
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
