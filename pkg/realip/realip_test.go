package realip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func req(remote string, hdr map[string]string) *http.Request {
	r := httptest.NewRequest("GET", "/wp-login.php", nil)
	r.RemoteAddr = remote
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	return r
}

func TestResolve(t *testing.T) {
	res, err := New([]string{"172.18.0.0/16", "192.0.2.10"}, "")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, remote, wantAddr, wantProxy string
		hdr                               map[string]string
	}{
		{"untrusted peer keeps its address, headers ignored", "203.0.113.9:4444", "203.0.113.9:4444", "",
			map[string]string{"X-Real-Ip": "198.51.100.1"}},
		{"trusted peer, X-Real-Ip", "172.18.0.1:5555", "198.51.100.1", "172.18.0.1",
			map[string]string{"X-Real-Ip": "198.51.100.1"}},
		{"trusted peer, X-Forwarded-For right-most untrusted", "172.18.0.1:5555", "198.51.100.7", "172.18.0.1",
			map[string]string{"X-Forwarded-For": "1.2.3.4, 198.51.100.7, 192.0.2.10"}},
		{"trusted peer, no header", "172.18.0.1:5555", "172.18.0.1:5555", "", nil},
		{"trusted peer, garbage header", "172.18.0.1:5555", "172.18.0.1:5555", "",
			map[string]string{"X-Real-Ip": "not-an-ip"}},
		{"bare trusted IP", "192.0.2.10:80", "198.51.100.2", "192.0.2.10",
			map[string]string{"X-Real-Ip": "198.51.100.2"}},
		{"IPv6 client", "172.18.0.1:5555", "2001:db8::1", "172.18.0.1",
			map[string]string{"X-Real-Ip": "2001:db8::1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := res.Resolve(req(c.remote, c.hdr))
			if got.RemoteAddr != c.wantAddr {
				t.Errorf("RemoteAddr = %q, want %q", got.RemoteAddr, c.wantAddr)
			}
			if p := ProxyAddr(got); p != c.wantProxy {
				t.Errorf("ProxyAddr = %q, want %q", p, c.wantProxy)
			}
		})
	}
}

func TestHeaderOrder(t *testing.T) {
	res, _ := New([]string{"172.18.0.0/16"}, "CF-Connecting-IP, X-Real-Ip")
	r := res.Resolve(req("172.18.0.1:1", map[string]string{"CF-Connecting-IP": "198.51.100.3", "X-Real-Ip": "162.158.1.1"}))
	if r.RemoteAddr != "198.51.100.3" {
		t.Errorf("RemoteAddr = %q, want the CF-Connecting-IP client", r.RemoteAddr)
	}
}

func TestDisabled(t *testing.T) {
	var res *Resolver
	r := req("172.18.0.1:1", map[string]string{"X-Real-Ip": "198.51.100.1"})
	if got := res.Resolve(r); got != r {
		t.Error("nil resolver must return the request unchanged")
	}
	if _, err := New([]string{"nope"}, ""); err == nil {
		t.Error("invalid CIDR must fail")
	}
}
