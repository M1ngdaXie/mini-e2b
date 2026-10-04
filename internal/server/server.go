package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/M1ngdaXie/mini-e2b/internal/fc"
)

/*
POST   /vms                    创建并启动，返回 {"id": "..."}
GET    /vms                    列出所有 VM
GET    /vms/{id}               查看一个
PATCH  /vms/{id}/state         暂停 / 恢复
POST   /vms/{id}/snapshots     打快照
DELETE /vms/{id}               停止
*/
type ListResponse struct {
	Id    string `json:"id"`
	State string `json:"state"`
}
type CreateResponse struct {
	Id string `json:"id"`
}
type Server struct {
	Addr       string
	Handler    http.Handler
	vms        *smap
	httpServer *http.Server
	wg         *sync.WaitGroup
}

func NewServer(addr string, handler http.Handler) *Server {
	s := &Server{
		Addr:    addr,
		Handler: handler,
		vms:     NewSmap(),
		httpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		wg: &sync.WaitGroup{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	mux.HandleFunc("GET /vms", s.handleListVms)
	mux.HandleFunc("POST /vms", s.handleCreateVm)
	mux.HandleFunc("DELETE /vms/{id}", s.handleDeleteVm)

	s.Handler = mux
	s.httpServer.Handler = s.Handler
	return s
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) handleListVms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vms := s.vms.List()
	resp := make([]ListResponse, 0, len(vms))
	for _, vm := range vms {
		resp = append(resp, ListResponse{
			Id:    vm.SandboxID,
			State: "Starting",
		})
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func (s *Server) handleCreateVm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vm, err := fc.NewVM()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-vm.Done()
		err := vm.ExitErr()
		if err != nil {
			log.Printf("Exit error from handleCreateVm: %v", err)
		}
		log.Printf("HandleCreateVm started the stoping process")
		cleanupErr := vm.Cleanup()
		if cleanupErr != nil {
			log.Printf("Error stopping VM : %v", cleanupErr)
		}
		s.vms.Remove(vm.SandboxID)
	}()
	err = vm.Boot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error Creating VM, stopping")
		stopErr := vm.Stop()
		if stopErr != nil {
			log.Printf("Error Stopping VM: %v", stopErr)
		}
		return
	}
	s.vms.Add(vm)

	resp := CreateResponse{
		Id: vm.SandboxID,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDeleteVm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	vm, ok := s.vms.Get(id)
	if !ok {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}
	log.Printf("HandleDeleteVm started the stoping process")
	err := vm.Stop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//wait for Close channel to be closed, then remove the VM from the store
	<-vm.Done()
	s.vms.Remove(id)
	w.Write([]byte("VM is Really gone"))
}

func (s *Server) HandleShutdown(ctx context.Context) error {
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
		return err
	}
	vmslist := s.vms.List()
	for _, vm := range vmslist {
		vm.Stop()
		<-vm.Done()
		log.Printf("VM stopped: %v", vm.SandboxID)
		s.vms.Remove(vm.SandboxID)
	}
	log.Printf("Signal sent to all VMs, waiting for WaitGroup to finish")
	s.wg.Wait()
	return nil
}
