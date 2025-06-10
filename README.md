# dns01proxy package for Caddy

[**dns01proxy**](https://github.com/liujed/dns01proxy) is a server for using
DNS-01 challenges to get TLS/SSL certificates from Let's Encrypt, or any
ACME-compatible certificate authority, without exposing your DNS credentials to
every host that needs a certificate.

This repository hosts and documents the dns01proxy Caddy package. If you want
to use dns01proxy as part of [Caddy](https://caddyserver.com/), then you're in
the right place! Otherwise, the [dns01proxy
project](https://github.com/liujed/dns01proxy) distributes a standalone
dns01proxy server as a precompiled binary, which is recommended for most users.

## Installing

This dns01proxy Caddy package provides:
* a `dns01proxy` command for running Caddy as a standalone dns01proxy server,
* an `http.handlers` module that implements the dns01proxy API, and
* a Caddy application module that implements the dns01proxy server.

To use this package, it needs to be compiled into your Caddy binary, along with
the [`dns.provider.*` modules](https://caddyserver.com/docs/modules/) for your
DNS providers. You can build your custom Caddy binary by using
[`xcaddy`](https://caddyserver.com/docs/build#xcaddy) or by using [Caddy's
download page](https://caddyserver.com/download).


## Using the command line

This package adds a `dns01proxy` command to Caddy, making it convenient to run
a standalone dns01proxy server. Naturally, because dns01proxy runs on Caddy, it
automatically obtains and renews its own TLS/SSL certificates using the
configured DNS credentials.

To run dns01proxy, just provide a config file:
```
caddy dns01proxy --config dns01proxy.toml
```

The example below configures dns01proxy for running at
`https://dns01proxy.example.com` with Cloudflare as a DNS provider.

```toml
hostnames = ["dns01proxy.example.com"]
listen = [":443"]

[dns.provider]
name = "cloudflare"
api_token = "{env.CF_API_TOKEN}"  # Reads from an environment variable.

# One for each user. Password is hashed using `caddy hash-password` with the
# bcrypt algorithm.
[[accounts]]
username = "AzureDiamond"
password = "$2a$14$N5bGBXf7zwAW9Ym7IQ/mxOHTGsvFNOTEAiN4/r1LnvfzYCpiWcHOa"
allow_domains = ["private.example.com"]
```

<details>
<summary>Full structure</summary>

```toml
# The server's hostnames. Used for obtaining TLS/SSL certificates.
hostnames = ["<hostname>"]

# The sockets on which to listen.
listen = ["<ip_addr:port>"]

# Configures the set of trusted proxies, for accurate logging of client IP
# addresses. This must be an `http.ip_sources` Caddy module. See Caddy's module
# documentation at https://caddyserver.com/docs/modules/
#
# Note that Caddy documents its modules' options in JSON. You'll need to
# configure the module in TOML. For example, to configure
# `http.ip_sources.static`:
#
#     [trusted_proxies]
#     source = "static"
#     ranges = ["10.0.0.1", "192.168.0.1"]
#
[trusted_proxies]
source = "<module_name>"
# •••  # Module-specific configuration goes here.

[dns]
# The TTL to use in DNS TXT records. Optional. Not usually needed.
ttl = "<ttl>"  # e.g., "2m"

# Custom DNS resolvers to prefer over system or built-in defaults. Set this to
# a public resolver if you are using split-horizon DNS.
resolvers = ["<resolver>"]

# The DNS provider for publishing DNS-01 responses. This must be a
# `dns.providers` Caddy module. See Caddy's module documentation at
# https://caddyserver.com/docs/modules/
#
# Note that Caddy documents its modules' options in JSON. You'll need to
# configure the module in TOML. For example, to configure
# `dns.providers.cloudflare`:
#
#     [dns.provider]
#     name = "cloudflare"
#     api_token = "{env.CF_API_TOKEN}"  # Reads from an environment variable.
#
[dns.provider]
name = "<provider_name>"
# •••  # Module-specific configuration goes here.


# Configures HTTP basic authentication and the domains for which each user can
# get TLS/SSL certificates.
[[accounts]]
user_id = "<userID>"
password = "<hashed_password>"  # To hash passwords, use `caddy hash-password`.

# These largely follow Smallstep's domain name rules:
#
#   https://smallstep.com/docs/step-ca/policies/#domain-names
#
# Due to a limitation in ACME and DNS-01, allowing a domain also allows
# wildcard certificates for that domain.
allow_domains = ["<domain>"]
deny_domains = ["<domain>"]
```

</details>

If you prefer JSON, you can use the same JSON structure as the configuration
for the [`dns01proxy` Caddy app](#configuring-a-dns01proxy-app-in-json).

## Integrating into a Caddyfile

This package provides the following Caddyfile handler directive.
```
dns01proxy {
  # The DNS provider for publishing DNS-01 responses. Optional. If this is
  # omitted, then the global `acme_dns` and `dns` options are used as
  # fallbacks, but at least one of the three must be configured.
  dns <provider_name> [<params...>]

  # The TTL to use in DNS TXT records. Optional. Not usually needed.
  dns_ttl <ttl>

  # Custom DNS resolvers to prefer over system or built-in defaults. Set this
  # to a public resolver if you are using split-horizon DNS.
  resolvers <resolvers...>

  # Configures a single user. Can be given multiple times.
  user <userID> {
    # Configures HTTP basic authentication for the user. This is optional. If
    # this is omitted, then an authentication handler must come before this one
    # in the handler chain. To hash passwords, use `caddy hash-password` with
    # the bcrypt algorithm.
    password <hashed_password>

    # Determines the domains for which the user can get TLS/SSL certificates.
    # This largely follows Smallstep's domain name rules:
    #
    #   https://smallstep.com/docs/step-ca/policies/#domain-names
    #
    # Due to a limitation in ACME and DNS-01, allowing a domain also allows
    # wildcard certificates for that domain.
    allow_domains <domains...>
    deny_domains <domains...>
  }
}
```

Here is an example Caddyfile for running dns01proxy as a standalone server that
automatically obtains and renews its own TLS/SSL certificate:
```
{
  acme_dns cloudflare {env.CF_API_TOKEN}
  cert_issuer acme {
    disable_http_challenge
    disable_tlsalpn_challenge
  }
}

dns01proxy.example.com {
  log
  @endpoints {
    path /present /cleanup
  }
  handle @endpoints {
    dns01proxy {
      user AzureDiamond {
        password $2a$14$N5bGBXf7zwAW9Ym7IQ/mxOHTGsvFNOTEAiN4/r1LnvfzYCpiWcHOa
        allow_domains private.example.com
      }
    }
  }
  respond 404
}
```

## Configuring a dns01proxy handler in JSON

The package also provides a `dns01proxy` HTTP handler. This example configures
a handler similar to the one in the Caddyfile example above.
```json
{
  "dns": {
    "provider": {
      "name": "cloudflare",
      "api_token": "{env.CF_API_TOKEN}"
    }
  },
  "accounts": [
    {
      "user_id": "AzureDiamond",
      "password": "$2a$14$N5bGBXf7zwAW9Ym7IQ/mxOHTGsvFNOTEAiN4/r1LnvfzYCpiWcHOa",
      "allow_domains": ["private.example.com"],
    }
  ]
}
```

<details>
<summary>Full JSON structure</summary>

```jsonc
{
  "dns": {
    // The DNS provider for publishing DNS-01 responses.
    "provider": {
      // a dns.providers module
      "name": "<provider_name>",
      // ••• 
    },

    // The TTL to use in DNS TXT records. Optional. Not usually needed.
    "ttl": "<ttl>",  // e.g., "2m"

    // Custom DNS resolvers to prefer over system or built-in defaults. Set
    // this to a public resolver if you are using split-horizon DNS.
    "resolvers": ["<resolver>"]
  },

  // Configures HTTP basic authentication (optional) and the domains for which
  // each user can get TLS/SSL certificates.
  //
  // Passwords are optional here. If they are omitted, then an authentication
  // handler must come before this one in the handler chain. To hash passwords,
  // use `caddy hash-password` with the bcrypt algorithm.
  "accounts": [
    {
      "user_id": "<userID>",
      "password": "<hashed_password>",
      "allow_domains": ["<domain>"],
      "deny_domains": ["<domain>"]
    }
  ]
}
```

</details>

## Configuring a dns01proxy app in JSON

Here is a sample configuration for the dns01proxy Caddy app, analogous to the
TOML example for the [`dns01proxy` command](#using-the-command-line).

```json
{
  "hostnames": ["dns01proxy.example.com"],
  "listen": [":443"],
  "dns": {
    "provider": {
      "name": "cloudflare",
      "api_token": "{env.CF_API_TOKEN}"
    }
  },
  "accounts": [
    {
      "username": "AzureDiamond",
      "password": "$2a$14$N5bGBXf7zwAW9Ym7IQ/mxOHTGsvFNOTEAiN4/r1LnvfzYCpiWcHOa",
      "allow_domains": ["private.example.com"]
    }
  ]
}
```

<details>
<summary>Full JSON structure</summary>

```jsonc
{
  // The server's hostnames. Used for obtaining TLS/SSL certificates.
  "hostnames": ["<hostname>"],

  // The sockets on which to listen.
  "listen": ["<ip_addr:port>"],

  // Configures the set of trusted proxies, for accurate logging of client IP
  // addresses.
  "trusted_proxies": {
    // an http.ip_sources module
    "source": "<module_name>",
    // •••
  },

  "dns": {
    // The DNS provider for publishing DNS-01 responses.
    "provider": {
      // A `dns.providers` module.
      "name": "<provider_name>",
      // ••• 
    },

    // The TTL to use in DNS TXT records. Optional. Not usually needed.
    "ttl": "<ttl>",  // e.g., "2m"

    // Custom DNS resolvers to prefer over system or built-in defaults. Set
    // this to a public resolver if you are using split-horizon DNS.
    "resolvers": ["<resolver>"]
  },

  // Configures HTTP basic authentication and the domains for which each user
  // can get TLS/SSL certificates.
  "accounts": [
    {
      "user_id": "<userID>",

      // To hash passwords, use `caddy hash-password`.
      "password": "<hashed_password>",

      // These largely follow Smallstep's domain name rules:
      //
      //   https://smallstep.com/docs/step-ca/policies/#domain-names
      //
      // Due to a limitation in ACME and DNS-01, allowing a domain also allows
      // wildcard certificates for that domain.
      "allow_domains": ["<domain>"],
      "deny_domains": ["<domain>"]
    }
  ]
}
```

</details>

## Acknowledgements

dns01proxy is a reimplementation of
[acmeproxy](https://github.com/mdbraber/acmeproxy/), which is no longer being
developed. Whereas acmeproxy was built on top of lego, dns01proxy uses
[libdns](https://github.com/libdns/libdns) under the hood, which allows for
better compatibility with acme.sh.

[acmeproxy.pl](https://github.com/madcamel/acmeproxy.pl) is another
reimplementation of acmeproxy, written in Perl.
