package httpapi

import "net/http"

// Serve uses HTTPS when a panel certificate and key are configured.
func Serve(server *http.Server, cert, key string) error {
	if cert != "" && key != "" {
		return server.ListenAndServeTLS(cert, key)
	}
	return server.ListenAndServe()
}
