package caddydns01proxy

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/liujed/goutil/maps"
)

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
