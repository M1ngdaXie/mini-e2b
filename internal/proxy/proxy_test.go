package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxy(t *testing.T) {
	var gotHost, gotXFF, gotheader string
	vm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotheader = r.Header.Get("X-vm-client")
		gotXFF = r.Header.Get("X-Forwarded-For")
		gotHost = r.Host
		w.Header().Set("X-Vm", "yes")
		w.Write([]byte("Hello from VM"))
	}))
	defer vm.Close()

	p, err := NewProxy(vm.URL)
	if err != nil {
		t.Fatalf("failed to start proxy:%v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-vm-client", "abc")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("code = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotheader != "abc" {
		t.Errorf("header = %s, want %s", gotheader, "abc")
	}
	if gotXFF != "192.0.2.1" {
		t.Errorf("got xff %s, want xff %s", gotXFF, "192.0.2.1")
	}
	if gotHost != vm.Listener.Addr().String() {
		t.Errorf("got %s, want %s", gotHost, vm.Listener.Addr().String())
	}
	if rec.Body.String() != "Hello from VM" {
		t.Errorf("body = %q", rec.Body)
	}
	if rec.Header().Get("X-Vm") != "yes" {
		t.Errorf("Wrong header")
	}
}

func TestServer502(t *testing.T) {
	backend := httptest.NewServer(nil)
	url := backend.URL
	backend.Close()
	p, err := NewProxy(url)
	if err != nil {
		t.Fatalf("failed to start proxy:%v", err)
	}
	req := httptest.NewRequest("GET", url, nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("got %d, want %d", rec.Code, http.StatusBadGateway)
	}
}
