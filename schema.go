package caddydns01proxy

// See https://github.com/libdns/acmeproxy/blob/f8e0a6620dddf349d1c9ba58b755aa7a25e5613f/provider.go#L20-L23.
type RequestBody struct {
	// The challenge domain at which the DNS-01 response should be written.
	ChallengeFQDN string `json:"fqdn"`

	// The value of the DNS-01 response.
	Value string `json:"value"`
}

func (r RequestBody) IsValid() bool {
	return r.ChallengeFQDN != "" && r.Value != ""
}

type ResponseBody = RequestBody
