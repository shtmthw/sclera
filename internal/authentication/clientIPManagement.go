package authentication

import (
	"net"
	"net/http"
	"strings"
)

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
