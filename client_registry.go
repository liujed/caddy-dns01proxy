package caddydns01proxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/liujed/goutil/maps"
	"github.com/liujed/goutil/optionals"
	"github.com/smallstep/certificates/policy"
)

type DenyReason string

const (
	// Indicates that authorization failed because the user's ID was not found in
	// the client registry.
	DenyUnknownUser DenyReason = "unknown user"

	// Indicates that authorization failed because the user is not authorized to
	// answer challenges for the requested domain.
	DenyDomainNotAllowed DenyReason = "requested domain denied by policy"

	// Indicates that authorization failed because the user requested an invalid
	// domain.
	DenyInvalidDomain DenyReason = "requested domain not valid"

	// Indicates that an error occurred during authorization.
	DenyError DenyReason = "an error occurred"
)

// DNS names for answering DNS-01 challenges are expected to have this prefix.
const challengeDomainPrefix = "_acme-challenge."

// A registry of known users and their corresponding policy configuration.
type ClientRegistry struct {
	// Maps each client's user ID to its policy configuration.
	Clients maps.Map[string, *ClientPolicy]
}

func (c *ClientRegistry) Provision(
	ctx caddy.Context,
	accountsRaw []RawAccount,
) error {
	// Convert accountsRaw into a map keyed on user ID.
	c.Clients = maps.NewHashMap[string, *ClientPolicy]()
	for i, rawAccount := range accountsRaw {
		if c.Clients.ContainsKey(rawAccount.UserID) {
			return fmt.Errorf(
				"account %d: user ID is not unique: %q",
				i,
				rawAccount.UserID,
			)
		}

		c.Clients.Put(rawAccount.UserID, &rawAccount.ClientPolicy)
	}

	// Provision the ClientPolicy instances.
	for userID, ca := range c.Clients.Entries() {
		err := ca.Provision(ctx)
		if err != nil {
			return fmt.Errorf(
				"unable to provision client policy for user ID %q: %w",
				userID,
				err,
			)
		}
	}

	return nil
}

// Determines whether the current authenticated user is allowed to answer a
// DNS-01 challenge at the given challenge domain. Returns None on success.
// Otherwise, returns the reason for denial.
func (r *ClientRegistry) AuthorizeUserChallengeDomain(
	req *http.Request,
	challengeDomain string,
) (optionals.Optional[DenyReason], error) {
	// Get the authenticated user ID from the context.
	repl := req.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	userID, exists := repl.GetString("http.auth.user.id")
	if !exists {
		// Authentication not configured?
		return optionals.Some(DenyError),
			fmt.Errorf("unable to determine user ID (is authentication configured?)")
	}

	config, exists := r.Clients.Get(userID).Get()
	if !exists {
		return optionals.Some(DenyUnknownUser), nil
	}

	// Deny if the challenge domain doesn't have the expected prefix.
	if !strings.HasPrefix(challengeDomain, challengeDomainPrefix) {
		return optionals.Some(DenyInvalidDomain), nil
	}

	// Strip off the prefix and remove any trailing dot. If the result starts with
	// a dot, then the requested domain is invalid. Otherwise, check the result
	// against the domain policy.
	domain := strings.TrimPrefix(challengeDomain, challengeDomainPrefix)
	domain = strings.TrimSuffix(domain, ".")
	if strings.HasPrefix(domain, ".") {
		return optionals.Some(DenyInvalidDomain), nil
	}
	err := config.DomainPolicy.IsDNSAllowed(domain)
	if err != nil {
		if npe, ok := err.(*policy.NamePolicyError); ok {
			switch npe.Reason {
			case policy.NotAllowed:
				return optionals.Some(DenyDomainNotAllowed), nil
			case policy.CannotParseDomain:
				return optionals.Some(DenyInvalidDomain), nil
			}
		}
		return optionals.Some(DenyError),
			fmt.Errorf("unable to authorize challenge domain: %w", err)
	}

	return optionals.None[DenyReason](), nil
}
