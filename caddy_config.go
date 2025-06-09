package caddydns01proxy

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/liujed/caddy-dns01proxy/jsonutil"
	"github.com/liujed/goutil/ptr"
)

// A dns01proxy configuration file is the same as the app configuration.
type ConfigFile = App

const defaultListen = "127.0.0.1:9095"

// Reads a dns01proxy configuration file and returns a corresponding Caddy
// configuration.
func caddyConfigFromConfigFile(path string) (*caddy.Config, error) {
	var config ConfigFile
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		// Convert TOML to JSON.
		rawMap := map[string]any{}
		_, err = toml.DecodeFile(path, &rawMap)
		if err != nil {
			return nil, err
		}
		rawJSON, err := json.Marshal(rawMap)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(rawJSON, &config)
		if err != nil {
			return nil, err
		}

	default:
		config, err = jsonutil.UnmarshalFromFile[ConfigFile](path)
		if err != nil {
			return nil, err
		}
	}

	// Set default listen sockets.
	if len(config.Listen) == 0 {
		config.Listen = []string{defaultListen}
	}

	appsRaw := caddy.ModuleMap{
		"dns01proxy": caddyconfig.JSON(config, nil),

		// Configure TLS automation to use the DNS provider.
		"tls": caddyconfig.JSON(config.MakeTLSConfig(), nil),
	}

	return &caddy.Config{
		Admin: &caddy.AdminConfig{
			Disabled: true,
			Config: &caddy.ConfigSettings{
				Persist: ptr.Of(false),
			},
		},
		AppsRaw: appsRaw,
	}, nil
}
