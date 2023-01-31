package helper

import "sync"

type NamedWaitGroup struct {
	m        map[string]int
	done     chan struct{}
	idx      int
	doneOnce map[int]chan string
	lock     sync.Mutex
}

func NewNamedWaitGroup() *NamedWaitGroup {
	return &NamedWaitGroup{
		m:        make(map[string]int),
		done:     make(chan struct{}, 0),
		idx:      0,
		doneOnce: make(map[int]chan string),
		lock:     sync.Mutex{},
	}
}

func (wg *NamedWaitGroup) Go(name string, f func()) {
	wg.Add(name)
	go func() {
		defer wg.Done(name)
		f()
	}()
}

func (wg *NamedWaitGroup) Add(name string) {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	if len(wg.m) == 0 {
		wg.done = make(chan struct{}, 1)
	}

	n, ok := wg.m[name]
	if !ok {
		n = 0
	}

	wg.m[name] = n + 1
}

func (wg *NamedWaitGroup) Done(name string) {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	n, ok := wg.m[name]
	if !ok {
		return
	}

	n--
	wg.m[name] = n
	if n > 0 {
		return
	}

	for seqNo, done := range wg.doneOnce {
		done <- name
		delete(wg.doneOnce, seqNo)
	}

	delete(wg.m, name)

	if len(wg.m) == 0 {
		close(wg.done)
	}
}

func (wg *NamedWaitGroup) Wait() {
	wg.lock.Lock()
	done := wg.done
	wg.lock.Unlock()

	<-done
}

func (wg *NamedWaitGroup) WaitOneName() (doneName string, active bool) {
	var idx int
	done := make(chan string, 0)
	{
		wg.lock.Lock()
		wg.idx++
		idx = wg.idx
		wg.doneOnce[idx] = done
		wg.lock.Unlock()
	}

	doneName = <-done

	wg.lock.Lock()
	delete(wg.doneOnce, idx)
	active = len(wg.m) > 0
	wg.lock.Unlock()

	return
}

func (wg *NamedWaitGroup) WaitOne() (active bool) {
	_, active = wg.WaitOneName()
	return
}

func (wg *NamedWaitGroup) ActiveNames() []string {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	names := make([]string, 0, len(wg.m))
	for name := range wg.m {
		names = append(names, name)
	}
	return names
}
