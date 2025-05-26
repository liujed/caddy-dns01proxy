package caddydns01proxy

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	x509policy "github.com/smallstep/certificates/authority/policy"
	"github.com/smallstep/certificates/authority/provisioner"
)

// The policy configuration for a user. Specifies the domains at which the user
// is allowed to answer DNS-01 challenges.
type ClientPolicy struct {
	// Identifies the client to which this policy applies.
	UserID string `json:"user_id"`

	AllowDomainsRaw []string `json:"allow_domains,omitempty"`
	DenyDomainsRaw  []string `json:"deny_domains,omitempty"`

	// The policy to be applied to the DNS domains for answering DNS-01
	// challenges.
	DomainPolicy x509policy.X509Policy `json:"-"`
}

var _ caddy.Provisioner = (*ClientPolicy)(nil)

func (c *ClientPolicy) Provision(ctx caddy.Context) error {
	domainPolicyOpts := provisioner.X509Options{}

	provisionX509NameOptions := func(raw *[]string) *x509policy.X509NameOptions {
		// Allow the raw version to be GC'd.
		defer func() {
			*raw = nil
		}()

		if len(*raw) > 0 {
			return &x509policy.X509NameOptions{
				DNSDomains: *raw,
			}
		}
		return nil
	}

	// Ingest the allow/deny lists.
	domainPolicyOpts.AllowedNames = provisionX509NameOptions(&c.AllowDomainsRaw)
	domainPolicyOpts.DeniedNames = provisionX509NameOptions(&c.DenyDomainsRaw)

	// Instantiate the domain policy.
	var err error
	c.DomainPolicy, err = x509policy.NewX509PolicyEngine(&domainPolicyOpts)
	if err != nil {
		return fmt.Errorf("unable to provision domain policy: %w", err)
	}

	return nil
}
