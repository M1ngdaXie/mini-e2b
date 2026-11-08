package network

import (
	"errors"
	"fmt"
	"log"
)

var ErrPoolEmpty = errors.New("pool is empty")

type Pool struct {
	poolChan chan int
}

func NewPool(n int) *Pool {
	pool := &Pool{
		poolChan: make(chan int, n),
	}
	for i := range n {
		pool.poolChan <- i
	}
	return pool
}

func (p *Pool) Acquire() (int, error) {
	select {
	case ip := <-p.poolChan:
		return ip, nil
	default:
		return -1, ErrPoolEmpty
	}
}

func (p *Pool) Release(idx int) error {
	select {
	case p.poolChan <- idx:
		log.Printf("Successfully release to channel with number:%d", idx)
		return nil
	default:
		return fmt.Errorf("BIG : Unexpected release on pool")
	}
}
