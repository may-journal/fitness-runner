package checkkit

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// NetworkEnv names the variable through which the runner tells each check
// that GitHub's API host did not answer its probe: `unreachable:<host>`.
// Unset means the host answered or the run probed nothing.
const NetworkEnv = "FITNESS_NETWORK"

// unreachablePrefix starts NetworkEnv's value when the probe failed.
const unreachablePrefix = "unreachable:"

// ProbeTimeout bounds the runner's one probe of GitHub's API host.
const ProbeTimeout = 2 * time.Second

// Probe opens one connection to target's host and closes it, through the
// proxy the environment names for it, within ProbeTimeout. Through an HTTP
// or HTTPS proxy it asks the proxy to CONNECT and waits for its answer. A
// nil error means the host answered. A proxy of another kind, such as
// SOCKS, counts as answering, so the reads go ahead as before.
func Probe(target *url.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), ProbeTimeout)
	defer cancel()
	proxy, err := ProxyFor(target)
	switch {
	case err != nil:
		return err
	case proxy == nil:
		return dialOnce(ctx, hostPort(target))
	case proxy.Scheme != "http" && proxy.Scheme != "https":
		return nil
	}
	return tunnel(ctx, proxy, hostPort(target))
}

// ProxyFor is the proxy HTTPS_PROXY, HTTP_PROXY and NO_PROXY name for
// target, as Go's HTTP client and gh pick it, or nil for none.
func ProxyFor(target *url.URL) (*url.URL, error) {
	return http.ProxyFromEnvironment(&http.Request{URL: target})
}

// Unreachable is the host the runner's probe could not reach, from
// NetworkEnv, or empty when it answered or was not probed.
func Unreachable() string {
	host, ok := strings.CutPrefix(os.Getenv(NetworkEnv), unreachablePrefix)
	if !ok {
		return ""
	}
	return host
}

// UnreachableEnv is the NetworkEnv entry that names host as unreachable.
func UnreachableEnv(host string) string {
	return NetworkEnv + "=" + unreachablePrefix + host
}

// Offline is the info note for a check that skips its network reads,
// named by skipped, because the runner's probe found GitHub's API host
// unreachable; empty when the reads should go ahead.
func Offline(skipped string) string {
	host := Unreachable()
	if host == "" {
		return ""
	}
	return OfflineNote(host, skipped)
}

// OfflineNote says that host did not answer the probe, through which proxy,
// and what was skipped without connecting.
func OfflineNote(host, skipped string) string {
	via := "with no proxy set"
	if proxy, err := ProxyFor(&url.URL{Scheme: "https", Host: host}); err == nil && proxy != nil {
		via = "through proxy " + proxy.Redacted()
	}
	return fmt.Sprintf("%s did not answer within %s %s, so fitness skipped %s without connecting", host, ProbeTimeout, via, skipped)
}

// dialOnce connects to addr and closes the connection at once.
func dialOnce(ctx context.Context, addr string) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}

// tunnel asks proxy to CONNECT to addr and waits for a 200 answer until
// ctx ends.
func tunnel(ctx context.Context, proxy *url.URL, addr string) error {
	conn, err := dialProxy(ctx, proxy)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	req := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: addr}, Host: addr, Header: proxyAuth(proxy)}
	if err := req.Write(conn); err != nil {
		return err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", proxy.Redacted(), resp.Status)
	}
	return nil
}

// dialProxy connects to proxy, over TLS for an HTTPS proxy.
func dialProxy(ctx context.Context, proxy *url.URL) (net.Conn, error) {
	if proxy.Scheme == "https" {
		return (&tls.Dialer{}).DialContext(ctx, "tcp", hostPort(proxy))
	}
	return (&net.Dialer{}).DialContext(ctx, "tcp", hostPort(proxy))
}

// proxyAuth is the header that signs in to proxy with the user its URL
// names, empty when it names none.
func proxyAuth(proxy *url.URL) http.Header {
	header := http.Header{}
	if user := proxy.User; user != nil {
		password, _ := user.Password()
		header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user.Username()+":"+password)))
	}
	return header
}

// hostPort is u's host and port, the scheme's default port when u names
// none.
func hostPort(u *url.URL) string {
	if u.Port() != "" {
		return u.Host
	}
	port := "443"
	if u.Scheme == "http" {
		port = "80"
	}
	return net.JoinHostPort(u.Hostname(), port)
}
