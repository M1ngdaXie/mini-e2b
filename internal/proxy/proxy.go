package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	target *url.URL
	pr     *httputil.ReverseProxy
}

func NewProxy(targetUrl string) (*Proxy, error) {
	t, err := url.Parse(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("Address is not valid:%w", err)
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(t)
			pr.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}
	return &Proxy{
		target: t,
		pr:     proxy,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.pr.ServeHTTP(w, r)
}
