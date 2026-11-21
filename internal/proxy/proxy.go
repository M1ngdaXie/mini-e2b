package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	lookup LookupFunc
	pr     *httputil.ReverseProxy
}
type LookupFunc func(id string) (ip string, ok bool)

func NewProxy(lookup LookupFunc) (*Proxy, error) {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			id := pr.In.PathValue("id")
			port := pr.In.PathValue("port")
			path := pr.In.PathValue("path")
			ip, ok := lookup(id)
			if !ok {
				return
			}
			u := fmt.Sprintf("http://%s:%s", ip, port)
			t, err := url.Parse(u)
			if err != nil {
				return
			}
			pr.SetURL(t)
			pr.Out.URL.Path = "/" + path
			pr.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}
	return &Proxy{
		lookup: lookup,
		pr:     proxy,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, ok := p.lookup(r.PathValue("id"))
	if !ok {
		http.Error(w, "cant find vm id, could be deleted", http.StatusNotFound)
		return
	}
	p.pr.ServeHTTP(w, r)
}
