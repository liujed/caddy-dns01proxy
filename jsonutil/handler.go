package jsonutil

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/liujed/goutil/optionals"
)

// Wraps the given handler function with a layer that handles the marshalling
// and unmarshalling of request/response modies.
func WrapHandler[RequestT any, ResponseT any](
	handler func(
		req *http.Request,
		requestBody RequestT,
	) (
		httpStatus int,
		responseBody optionals.Optional[ResponseT],
		err error,
	),
) caddyhttp.Handler {
	return wrappedHandler[RequestT, ResponseT]{handler: handler}
}

type wrappedHandler[RequestT any, ResponseT any] struct {
	handler func(
		req *http.Request,
		requestBody RequestT,
	) (
		httpStatus int,
		responseBody optionals.Optional[ResponseT],
		err error,
	)
}

var _ caddyhttp.Handler = (*wrappedHandler[any, any])(nil)

func (h wrappedHandler[RequestT, ResponseT]) ServeHTTP(
	w http.ResponseWriter,
	req *http.Request,
) error {
	// Unmarshal the request body.
	var reqBody RequestT
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		http.Error(
			w,
			"Unable to read request body as JSON",
			http.StatusBadRequest,
		)
		return nil
	}

	// Call the wrapped handler.
	httpStatus, responseBodyOpt, err := h.handler(req, reqBody)
	if err != nil {
		return err
	}

	// Write out the response.
	if respBody, exists := responseBodyOpt.Get(); exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		err = json.NewEncoder(w).Encode(respBody)
		if err != nil {
			return fmt.Errorf("unable to write response body: %w", err)
		}
		return nil
	}

	// No response body. Just write the HTTP status.
	w.WriteHeader(httpStatus)
	return nil
}
