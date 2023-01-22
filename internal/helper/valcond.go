package helper

import (
	"sync"
	"sync/atomic"
)

type ValCond[T comparable] struct {
	val  atomic.Value
	cond *sync.Cond
}

func NewValCond[T comparable](val T) *ValCond[T] {
	vc := ValCond[T]{cond: sync.NewCond(new(sync.Mutex))}
	vc.val.Store(val)
	return &vc
}

func (vc *ValCond[T]) Get() T {
	return vc.val.Load().(T)
}

func (vc *ValCond[T]) Set(val T) {
	vc.val.Store(val)
	vc.cond.Broadcast()
}

func (vc *ValCond[T]) Is(val T) bool {
	return vc.val.Load().(T) == val
}

func (vc *ValCond[T]) IsNot(val T) bool {
	return vc.val.Load().(T) != val
}

func (vc *ValCond[T]) WaitEq(val T) {
	for vc.IsNot(val) {
		vc.cond.Wait()
	}
}

func (vc *ValCond[T]) WaitNotEq(val T) {
	for vc.Is(val) {
		vc.cond.Wait()
	}
}
