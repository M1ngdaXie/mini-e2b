package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestMux(t *testing.T, vms map[string]string) *http.ServeMux {
	lookup := func(id string) (string, bool) {
		id, ok := vms[id]
		if !ok {
			return "", false
		}
		return id, true
	}
	p, err := NewProxy(lookup)
	if err != nil {
		t.Fatalf("failed to start proxy:%v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/vms/{id}/proxy/{port}/{path...}", p)
	return mux
}

// func TestProxy(t *testing.T) {
// 	var gotHost, gotXFF, gotheader string
// 	vm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		gotheader = r.Header.Get("X-vm-client")
// 		gotXFF = r.Header.Get("X-Forwarded-For")
// 		gotHost = r.Host
// 		w.Header().Set("X-Vm", "yes")
// 		w.Write([]byte("Hello from VM"))
// 	}))
// 	defer vm.Close()
//
// 	p, err := NewProxy(lookup)
// 	mux := http.NewServeMux()
// 	mux.Handle("/vms/{id}/proxy/{port}/{path...}", p)
// 	if err != nil {
// 		t.Fatalf("failed to start proxy:%v", err)
// 	}
// 	url404 := fmt.Sprintf("/vms/nope/proxy/%s/", port)
//
// 	req := httptest.NewRequest("GET", "/", nil)
// 	req403 := httptest.NewRequest("GET", url404, nil)
//
// 	req.Header.Set("X-vm-client", "abc")
// 	rec := httptest.NewRecorder()
// 	mux.ServeHTTP(rec, req)
// 	if rec.Code != http.StatusOK {
// 		t.Errorf("code = %d, want %d", rec.Code, http.StatusOK)
// 	}
// 	if gotheader != "abc" {
// 		t.Errorf("header = %s, want %s", gotheader, "abc")
// 	}
// 	if gotXFF != "192.0.2.1" {
// 		t.Errorf("got xff %s, want xff %s", gotXFF, "192.0.2.1")
// 	}
// 	if gotHost != vm.Listener.Addr().String() {
// 		t.Errorf("got %s, want %s", gotHost, vm.Listener.Addr().String())
// 	}
// 	if rec.Body.String() != "Hello from VM" {
// 		t.Errorf("body = %q", rec.Body)
// 	}
// 	if rec.Header().Get("X-Vm") != "yes" {
// 		t.Errorf("Wrong header")
// 	}
// }
//
// func TestServer502(t *testing.T) {
// 	backend := httptest.NewServer(nil)
// 	url := backend.URL
// 	backend.Close()
// 	p, err := NewProxy(url)
// 	if err != nil {
// 		t.Fatalf("failed to start proxy:%v", err)
// 	}
// 	req := httptest.NewRequest("GET", url, nil)
// 	rec := httptest.NewRecorder()
// 	p.ServeHTTP(rec, req)
// 	if rec.Code != http.StatusBadGateway {
// 		t.Errorf("got %d, want %d", rec.Code, http.StatusBadGateway)
// 	}
// }

func TestProxyOK(t *testing.T) {
	vm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from VM"))
	}))
	defer vm.Close()

	host, port, err := net.SplitHostPort(vm.Listener.Addr().String())
	if err != nil {
		t.Fatalf("can not split test vm addres:%v", err)
	}
	fakeVMs := map[string]string{
		"abc": host,
	}
	mux := newTestMux(t, fakeVMs)
	url200 := fmt.Sprintf("/vms/abc/proxy/%s/", port)
	req200 := httptest.NewRequest("GET", url200, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req200)
	if rec.Code != 200 || rec.Body.String() != "Hello from VM" {
		t.Errorf("got %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
}

func TestProxyNotfound(t *testing.T) {
	vm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from VM"))
	}))
	defer vm.Close()
	host, port, err := net.SplitHostPort(vm.Listener.Addr().String())
	if err != nil {
		t.Fatalf("can not split test vm addres:%v", err)
	}
	fakeVMs := map[string]string{
		"abc": host,
	}
	mux := newTestMux(t, fakeVMs)
	url404 := fmt.Sprintf("/vms/nope/proxy/%s/", port)
	req403 := httptest.NewRequest("GET", url404, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req403)
	if rec.Code != 404 {
		t.Errorf("got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProxyBadGateway(t *testing.T) {
	vm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from VM"))
	}))
	defer vm.Close()

	host, port, err := net.SplitHostPort(vm.Listener.Addr().String())
	_ = port
	if err != nil {
		t.Fatalf("can not split test vm addres:%v", err)
	}
	fakeVMs := map[string]string{
		"abc": host,
	}
	mux := newTestMux(t, fakeVMs)
	deadserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, %s", r.URL.Path)
	}))
	deadserver.Close()
	deadport := deadserver.Listener.Addr().(*net.TCPAddr).Port // int 类型的端口号
	url502 := fmt.Sprintf("/vms/abc/proxy/%d/", deadport)
	req502 := httptest.NewRequest("GET", url502, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req502)
	if rec.Code != 502 {
		t.Errorf("got %d, want %d", rec.Code, http.StatusBadGateway)
	}
}
