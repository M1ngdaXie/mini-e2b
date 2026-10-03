package server

import (
	"encoding/json"
	"net/http"
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
	// mux.HandleFunc("POST /vms", handleOnlyVmCreate)

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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
