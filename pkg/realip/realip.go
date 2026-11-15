// Package realip resolves the client address of requests that reach galah
// through a trusted reverse proxy (Traefik, nginx, a CDN). Without it, every
// event is attributed to the proxy, and enrichment and sessions break down.
package realip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// DefaultHeaders is the header order used when none is configured. Both are
// set by the proxy itself (Traefik and nginx overwrite them for untrusted
// peers), so a client can't choose its own address through them.
var DefaultHeaders = []string{"X-Real-Ip", "X-Forwarded-For"}

type ctxKey struct{}

// Resolver rewrites RemoteAddr for requests from trusted proxies.
type Resolver struct {
	Trusted []*net.IPNet
	Headers []string
}

// New parses CIDRs or bare IPs into a Resolver. headers is a comma-separated
// list; empty means DefaultHeaders.
func New(trusted []string, headers string) (*Resolver, error) {
	res := &Resolver{Headers: DefaultHeaders}
	for _, t := range trusted {
		for _, item := range strings.Split(t, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if !strings.Contains(item, "/") {
				ip := net.ParseIP(item)
				if ip == nil {
					return nil, fmt.Errorf("invalid trusted proxy %q", item)
				}
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				item = fmt.Sprintf("%s/%d", ip.String(), bits)
			}
			_, n, err := net.ParseCIDR(item)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy %q: %w", item, err)
			}
			res.Trusted = append(res.Trusted, n)
		}
	}
	if h := strings.TrimSpace(headers); h != "" {
		res.Headers = nil
		for _, name := range strings.Split(h, ",") {
			if name = strings.TrimSpace(name); name != "" {
				res.Headers = append(res.Headers, name)
			}
		}
	}
	return res, nil
}

func (res *Resolver) trusted(ip net.IP) bool {
	for _, n := range res.Trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// Resolve returns r unchanged unless it came from a trusted proxy and carries
// a usable client address. Then it returns a shallow copy whose RemoteAddr is
// the client IP (no port: the proxy doesn't pass it) and whose context
// remembers the proxy's address (see ProxyAddr).
func (res *Resolver) Resolve(r *http.Request) *http.Request {
	if res == nil || len(res.Trusted) == 0 {
		return r
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil || !res.trusted(peer) {
		return r
	}
	client := res.clientFromHeaders(r.Header)
	if client == "" {
		return r
	}
	r2 := r.WithContext(context.WithValue(r.Context(), ctxKey{}, host))
	r2.RemoteAddr = client
	return r2
}

func (res *Resolver) clientFromHeaders(h http.Header) string {
	for _, name := range res.Headers {
		v := h.Get(name)
		if v == "" {
			continue
		}
		if !strings.Contains(v, ",") {
			if ip := net.ParseIP(strings.TrimSpace(v)); ip != nil {
				return ip.String()
			}
			continue
		}
		// A list (X-Forwarded-For): the right-most address that isn't one of
		// our trusted proxies is the last hop we can vouch for.
		parts := strings.Split(v, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				break
			}
			if !res.trusted(ip) {
				return ip.String()
			}
		}
	}
	return ""
}

// ProxyAddr returns the proxy address recorded by Resolve, or "".
func ProxyAddr(r *http.Request) string {
	v, _ := r.Context().Value(ctxKey{}).(string)
	return v
}
