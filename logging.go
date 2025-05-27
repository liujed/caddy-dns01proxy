package caddydns01proxy

import (
	"net/http"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

const (
	// Log key for reporting why a user failed authorization.
	logAuthorizationFailure = "deny_reason"
)

// Adds the given field to the access logs for the given request.
func addLogField(req *http.Request, field zap.Field) {
	extra := req.Context().Value(caddyhttp.ExtraLogFieldsCtxKey).(*caddyhttp.ExtraLogFields)
	extra.Add(field)
}
