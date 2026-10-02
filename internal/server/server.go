package server

import "net/http"

type Server struct {
	Addr    string
	Handler http.Handler
}

func NewServer(addr string, handler http.Handler) *Server {
	return &Server{
		Addr:    addr,
		Handler: handler,
	}
}

func (s *Server) Start() error {
	return http.ListenAndServe(s.Addr, s.Handler)
}
