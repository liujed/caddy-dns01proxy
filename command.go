package caddydns01proxy

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/liujed/caddy-dns01proxy/flags"
	"github.com/liujed/goutil/optionals"
	"github.com/spf13/cobra"
)

// Flag definitions.
var (
	flgConfig = flags.Flag[string]{
		Name:         "config",
		ShortName:    optionals.Some('c'),
		UsageMsg:     "read configuration from `FILE`",
		Required:     true,
		FilenameExts: optionals.Some([]string{"json"}),
	}

	flgDebug = flags.Flag[bool]{
		Name:      "debug",
		ShortName: optionals.Some('v'),
		UsageMsg:  "turn on verbose debug logs",
	}
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "dns01proxy",
		Short: "Starts a proxy server for responding to DNS-01 challenges",
		Long: `
dns01proxy is a server for using DNS-01 challenges to obtain TLS certificates
from Let's Encrypt, or any ACME-compatible certificate authority, without
exposing your DNS credentials to every host that needs a certificate.

It acts as a proxy for DNS-01 challenge requests, allowing hosts to delegate
their DNS record updates during ACME validation. This makes it possible to issue
certificates to internal or private hosts that can't (or shouldn't) have direct
access to your DNS provider or API keys.

Designed to work with:
  * acme.sh's 'acmeproxy' provider,
  * Caddy's 'acmeproxy' DNS provider module, and
  * lego's 'httpreq' DNS provider.`,
		CobraFunc: func(cmd *cobra.Command) {
			flags.AddStringFlag(cmd, flgConfig)
			flags.AddBoolFlag(cmd, flgDebug)

			cmd.RunE = caddycmd.WrapCommandFuncForCobra(cmdRun)

			cmd.AddCommand(&cobra.Command{
				Use:   "version",
				Short: "Print version information",
				Run: func(*cobra.Command, []string) {
					fmt.Println(Release())
				},
			})
		},
	})
}

func cmdRun(fs caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	configFlag := fs.String(flgConfig.Name)
	cfg, err := caddyConfigFromConfigFile(configFlag)
	if err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	caddy.Log().Info(fmt.Sprintf("Starting %s", Release()))

	err = caddy.Run(cfg)
	if err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	select {}
}
