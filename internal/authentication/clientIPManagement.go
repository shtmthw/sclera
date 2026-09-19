package authentication

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
)

// GetClientIP returns the original client IP when the direct peer is
// a trusted reverse proxy on the Docker network.

var ErrInvalidIP = errors.New("IP is INVAID/UNAUTHORIZED")

func GetClientIP(r *http.Request, trustedProxyNet *net.IPNet) (string, error) {

	// Extract the remote ip, must be of the Nginx server
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("error while splitting host ip")
		return "", err
	}

	// string to []byte conversion of the ip string
	remoteIP := net.ParseIP(remoteHost)

	// Check if the remote IP is in the trusted proxy net
	if trustedProxyNet == nil || !trustedProxyNet.Contains(remoteIP) {
		log.Printf("remoteIP is not in trusted proxy net or empty")
		return "", ErrInvalidIP
	}

	// process the actual user ip.
	xrip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
	if xrip == nil {
		log.Printf("XRIP is nil")
		return "", ErrInvalidIP
	}

	return xrip.String(), nil
}
