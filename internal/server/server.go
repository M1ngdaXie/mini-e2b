package server

import (
	"encoding/json"
	"log"
	"net/http"

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
	Addr    string
	Handler http.Handler
	vms     *smap
}

func NewServer(addr string, handler http.Handler) *Server {
	s := &Server{
		Addr:    addr,
		Handler: handler,
		vms:     NewSmap(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	mux.HandleFunc("GET /vms", s.handleListVms)
	mux.HandleFunc("POST /vms", s.handleCreateVm)
	mux.HandleFunc("DELETE /vms/{id}", s.handleDeleteVm)

	s.Handler = mux
	return s
}

func (s *Server) Start() error {
	return http.ListenAndServe("127.0.0.1:"+s.Addr, s.Handler)
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
	go func() {
		<-vm.Done()
		err := vm.ExitErr()
		if err != nil {
			log.Printf("Exit error: %v", err)
		}
		stopErr := vm.Stop()
		if stopErr != nil {
			log.Printf("Error stopping VM : %v", stopErr)
		}
		s.vms.Remove(vm.SandboxID)
	}()

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
	err := vm.Stop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.vms.Remove(id)
	w.Write([]byte("VM deleted"))
}

// func handleOnlyVmCreate(w http.ResponseWriter, r *http.Request) {
// 	if r.Method == "POST" {
// 		vm, err := NewVm()
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 		w.Write([]byte(vm.Id))
// 	}
// }
