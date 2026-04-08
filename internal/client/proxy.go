package client

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/proxy"
)

type proxyConfig struct {
	proxyURL       *url.URL
	noProxy        []string
	useEnvProxy    bool
	socks5ProxyURL *url.URL
}

func (c *Client) SetProxyURL(proxyURLStr string) {
	if c.proxyConfig == nil {
		c.proxyConfig = &proxyConfig{}
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return
	}

	switch proxyURL.Scheme {
	case "socks5":
		c.proxyConfig.socks5ProxyURL = proxyURL
		c.proxyConfig.proxyURL = nil
	default:
		c.proxyConfig.proxyURL = proxyURL
		c.proxyConfig.socks5ProxyURL = nil
	}

	c.applyProxyConfig()
}

func (c *Client) SetNoProxy(noProxyHosts []string) {
	if c.proxyConfig == nil {
		c.proxyConfig = &proxyConfig{}
	}
	c.proxyConfig.noProxy = noProxyHosts
	c.applyProxyConfig()
}

func (c *Client) UseEnvironmentProxy() {
	if c.proxyConfig == nil {
		c.proxyConfig = &proxyConfig{}
	}
	c.proxyConfig.useEnvProxy = true
	c.applyProxyConfig()
}

func (c *Client) applyProxyConfig() {
	if c.transport == nil {
		c.transport = &http.Transport{}
	}

	if c.proxyConfig == nil {
		c.transport.Proxy = nil
		return
	}

	if c.proxyConfig.socks5ProxyURL != nil {
		dialer, err := proxy.SOCKS5(
			c.proxyConfig.socks5ProxyURL.Scheme,
			c.proxyConfig.socks5ProxyURL.Host,
			nil,
			proxy.Direct,
		)
		if err == nil {
			c.transport.Dial = dialer.Dial
		}
		return
	}

	if c.proxyConfig.useEnvProxy {
		c.transport.Proxy = http.ProxyFromEnvironment
		return
	}

	if c.proxyConfig.proxyURL != nil {
		c.transport.Proxy = func(req *http.Request) (*url.URL, error) {
			if c.proxyConfig != nil && len(c.proxyConfig.noProxy) > 0 {
				host := req.URL.Hostname()
				for _, noProxy := range c.proxyConfig.noProxy {
					if noProxy == "*" || host == noProxy || strings.HasSuffix(host, "."+noProxy) {
						return nil, nil
					}
				}
			}
			return c.proxyConfig.proxyURL, nil
		}
		return
	}

	c.transport.Proxy = nil
}

func (c *Client) getProxyURL(req Request) *url.URL {
	if req.ProxyURL != "" {
		proxyURL, err := url.Parse(req.ProxyURL)
		if err == nil {
			return proxyURL
		}
	}
	if c.proxyConfig != nil && c.proxyConfig.proxyURL != nil {
		return c.proxyConfig.proxyURL
	}
	if c.proxyConfig != nil && c.proxyConfig.useEnvProxy {
		return nil
	}
	return nil
}

func isProxyAuth(proxyURL *url.URL) (string, string) {
	if proxyURL == nil {
		return "", ""
	}
	if proxyURL.User == nil {
		return "", ""
	}
	return proxyURL.User.Username(), ""
}

func shouldUseProxy(req *http.Request, noProxy []string) bool {
	if len(noProxy) == 0 {
		return true
	}
	host := req.URL.Hostname()
	for _, np := range noProxy {
		if np == "*" || host == np || strings.HasSuffix(host, "."+np) {
			return false
		}
	}
	return true
}

func (c *Client) buildClientWithProxy(req Request) *http.Client {
	transport := &http.Transport{
		TLSClientConfig:   c.transport.TLSClientConfig,
		DialContext:       c.transport.DialContext,
		DisableKeepAlives: c.transport.DisableKeepAlives,
	}

	if req.ProxyURL != "" {
		proxyURL, err := url.Parse(req.ProxyURL)
		if err == nil {
			if proxyURL.Scheme == "socks5" {
				dialer, err := proxy.SOCKS5("socks5", proxyURL.Host, nil, proxy.Direct)
				if err == nil {
					transport.Dial = dialer.Dial
				}
			} else {
				transport.Proxy = func(r *http.Request) (*url.URL, error) {
					noProxy := req.NoProxy
					if c.proxyConfig != nil && len(c.proxyConfig.noProxy) > 0 {
						noProxy = append(noProxy, c.proxyConfig.noProxy...)
					}
					if len(noProxy) > 0 {
						host := r.URL.Hostname()
						for _, np := range noProxy {
							if np == "*" || host == np || strings.HasSuffix(host, "."+np) {
								return nil, nil
							}
						}
					}
					return proxyURL, nil
				}
			}
		}
	} else if c.proxyConfig != nil {
		if c.proxyConfig.useEnvProxy {
			transport.Proxy = http.ProxyFromEnvironment
		} else if c.proxyConfig.proxyURL != nil {
			transport.Proxy = c.transport.Proxy
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   c.timeout,
	}
}
