package network

import (
	"testing"
)

func setup(n int) *Pool {
	p := NewPool(n)
	return p
}

func TestWithdrawTwice(t *testing.T) {
	p := NewPool(2)
	a, err := p.Acquire()
	if err != nil {
		t.Fatalf("first Acquire: unexpected error: %v", err) // 期望：没有错误
	}
	b, err := p.Acquire()
	if err != nil {
		t.Fatalf("second Acquire: unexpected error: %v", err)
	}
	if a == b {
		t.Fatalf("got the same index twice: %d", a)
	}
	c, err := p.Acquire()
	if err == nil {
		t.Fatalf("third Acquire: expected error on empty pool, got index %d", c)
	}
	p.Release(b)
	d, err := p.Acquire()
	if err != nil {
		t.Fatalf("Fourth Acquire: unexpected error: %v", err)
	}
	if d != b {
		t.Fatalf("Wrong number with Fourth acquire")
	}
}
func TestAcquire(t *testing.T) {
	p := setup(2)
	for i := range len(p.poolChan) + 1 {
		_ = i
		ip, err := p.Acquire()
		if err != nil {
			if i == 2 {
				continue
			}
			t.Fatalf("Error testing acquire:%v ", err)
		}
		t.Logf("Acquire success : %d", ip)
	}
}

func TestRelease(t *testing.T) {
	p := setup(2)
	for i := range len(p.poolChan) {
		_ = i
		ip, err := p.Acquire()
		if err != nil {
			t.Fatalf("Error testing acquire:%v ", err)
		}
		t.Logf("Acquire success : %d", ip)
	}
	p.Release(9)
	ip, err := p.Acquire()
	if err != nil {
		t.Fatalf("Error testing acquire:%v ", err)
		return
	}
	t.Logf("Acquire success : %d", ip)

}

func TestReleaseFull(t *testing.T) {
	p := setup(3)
	err := p.Release(9)
	if err == nil {
		t.Fatalf("Unexpected accept on full channel")
	}
}
