package chat

import (
	"context"
	gohttp "net/http"

	"github.com/tonto/kit/http"
)

// WithHTTPBasicAuth creates new basic auth adapter
// that checks for provided admin username and password
func WithHTTPBasicAuth(uname, pass string) http.Adapter {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(c context.Context, w gohttp.ResponseWriter, r *gohttp.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || (u != uname || p != pass) {
				w.Header().Add("WWW-Authenticate", `Basic realm="Access to chat api"`)
				w.WriteHeader(gohttp.StatusUnauthorized)
				return
			}

			h(c, w, r)
		}
	}
}
