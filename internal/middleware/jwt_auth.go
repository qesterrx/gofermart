package middleware

import (
	"context"
	"net/http"

	"github.com/qesterrx/gofermart/internal/auth"
)

// JWTAccess - middleware проверяющий наличие cookie авторизации и добавляющий из него ИД пользователя в контекст *http.Request
func JWTAccess(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.JWTCookieName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		jwtc, err := auth.ValidateToken(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), auth.JWTContextKeys, jwtc)
		h.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}
