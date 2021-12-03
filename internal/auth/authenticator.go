package auth

import (
	"context"
	"errors"
	"net/http"
)

func FromContext(ctx context.Context) (*JWT, map[string]interface{}, error) {
	token, ok := ctx.Value("Token").(JWT)

	var err error
	var claims map[string]interface{}

	if ok {
		claims = token.Payload
	} else {
		claims = map[string]interface{}{}
		return &token, claims, errors.New("token not presen")
	}

	err, _ = ctx.Value("Error").(error)

	return &token, claims, err
}

func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := FromContext(r.Context())

		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if token == nil || token.Validate() != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// Token is authenticated, pass it through
		next.ServeHTTP(w, r)
	})
}
