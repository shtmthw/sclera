package authentication

import (
	"net"
	"net/http"
	"strings"
)

// IsTrustedProxy reports whether the direct TCP peer is the verified nginx
// reverse proxy on our Docker network (trustedProxyNet, e.g. 172.19.0.0/16).
// Strict: loopback is NOT trusted — direct :8080 hits (even from localhost)
// are blocked. Use the nginx entrypoint (127.0.0.1:81) for dev.
func IsTrustedProxy(r *http.Request, trustedProxyNet *net.IPNet) bool {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	remoteIP := net.ParseIP(remoteHost)
	if remoteIP == nil {
		return false
	}

	return trustedProxyNet != nil && trustedProxyNet.Contains(remoteIP)
}

// WriteProxyBlockHTML rejects a request that did not arrive via the verified
// proxy with a red-background / black-text page.
func WriteProxyBlockHTML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`<!DOCTYPE html><html><body style="background:red;color:black;font-family:monospace;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;"><h1>NICE TRY SON</h1></body></html>`))
}

// GetClientIP returns the original client IP when the direct peer is
// a trusted reverse proxy on the Docker network.
func GetClientIP(r *http.Request, trustedProxyNet *net.IPNet) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	remoteIP := net.ParseIP(remoteHost)
	if remoteIP == nil {
		return remoteHost
	}

	// Only trust forwarded headers when the direct peer belongs
	// to our trusted Docker network.
	if trustedProxyNet != nil && trustedProxyNet.Contains(remoteIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")

			// Nginx is configured to overwrite X-Forwarded-For
			// with the actual client IP, so take the first value.
			candidate := strings.TrimSpace(parts[0])

			if ip := net.ParseIP(candidate); ip != nil {
				return ip.String()
			}
		}

		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			if ip := net.ParseIP(strings.TrimSpace(xrip)); ip != nil {
				return ip.String()
			}
		}
	}

	// Direct connection or untrusted proxy.
	return remoteIP.String()
}
