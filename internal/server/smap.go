package server

import (
	"log"
	"sync"

	"github.com/M1ngdaXie/mini-e2b/internal/fc"
)

type smap struct {
	// id -> VM
	m  map[string]*fc.VM
	mu sync.RWMutex
}

func NewSmap() *smap {
	return &smap{
		m: make(map[string]*fc.VM),
	}
}

func (s *smap) Get(id string) (*fc.VM, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vm, ok := s.m[id]
	return vm, ok
}

func (s *smap) set(id string, vm *fc.VM) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[id]
	if ok {
		log.Printf("vm already exists: %s", id)
		return
	}
	s.m[id] = vm
}

func (s *smap) Add(vm *fc.VM) {
	s.set(vm.SandboxID, vm)
}

func (s *smap) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

func (s *smap) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

func (s *smap) List() []*fc.VM {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vms := make([]*fc.VM, 0, len(s.m))
	for _, vm := range s.m {
		vms = append(vms, vm)
	}
	return vms
}
