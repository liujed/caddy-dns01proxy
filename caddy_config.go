package caddydns01proxy

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/liujed/caddy-dns01proxy/jsonutil"
)

// A dns01proxy configuration file is the same as the app configuration.
type ConfigFile = App

// Reads a dns01proxy configuration file and returns a corresponding Caddy
// configuration.
func caddyConfigFromConfigFile(path string) (*caddy.Config, error) {
	config, err := jsonutil.UnmarshalFromFile[ConfigFile](path)
	if err != nil {
		return nil, err
	}

	// TODO: generate a Caddy configuration.
	_ = config

	return nil, fmt.Errorf("implement me")
}
