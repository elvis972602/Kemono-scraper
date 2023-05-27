package downloader

import (
	"context"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func parseProxyUrl(proxyUrlStr string) (proxyType, proxyAddr string) {
	u, err := url.Parse(proxyUrlStr)
	if err != nil {
		panic(err)
	}
	proxyType = strings.ToLower(u.Scheme)
	proxyAddr = u.Host
	return
}

func AddProxy(proxyUrlStr string, transport *http.Transport) {
	proxyType, _ := parseProxyUrl(proxyUrlStr)
	proxyUrl, err := url.Parse(proxyUrlStr)
	if err != nil {
		panic(err)
	}
	switch proxyType {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyUrl)
	case "socks5":
		var dialer proxy.Dialer
		dialer, err = proxy.FromURL(proxyUrl, proxy.Direct)
		if err != nil {
			panic(err)
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	default:
		panic("unsupported proxy type: " + proxyType)
	}
}
