package middleware

import "net/http"

// JsonContentType - middleware проверяющий наличие заголовка "Content-Type":"application/json"
func JsonContentType(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" || r.ContentLength == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// TextContentType - middleware проверяющий наличие заголовка "Content-Type":"text/plain"
func TextContentType(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" || r.ContentLength == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
