package helper

// import (
// 	"sync"
// )

// type ValCond[T comparable] struct {
// 	val  T
// 	cond *sync.Cond
// 	mu   sync.Mutex
// }

// func NewValCond[T comparable](val T) *ValCond[T] {
// 	return &ValCond[T]{
// 		val:  val,
// 		cond: sync.NewCond(new(sync.Mutex)),
// 		mu:   sync.Mutex{},
// 	}
// }

// func (vc *ValCond[T]) Get() T {
// 	vc.mu.Lock()
// 	defer vc.mu.Unlock()
// 	return vc.val
// }

// func (vc *ValCond[T]) Set(val T) (oldVal T) {
// 	vc.mu.Lock()
// 	defer vc.mu.Unlock()
// 	oldVal = vc.val
// 	vc.val = val
// 	vc.cond.Broadcast()
// 	return vc.oldVal
// }

// func (vc *ValCond[T]) Is(val T) bool {
// 	vc.mu.Lock()
// 	defer vc.mu.Unlock()
// 	return vc.val.Load().(T) == val
// }

// func (vc *ValCond[T]) IsNot(val T) bool {
// 	vc.mu.Lock()
// 	defer vc.mu.Unlock()
// 	return vc.val.Load().(T) != val
// }

// func (vc *ValCond[T]) WaitEq(val T) {
// 	for vc.IsNot(val) {
// 		vc.cond.Wait()
// 	}
// }

// func (vc *ValCond[T]) WaitNotEq(val T) {
// 	for vc.Is(val) {
// 		vc.cond.Wait()
// 	}
// }
