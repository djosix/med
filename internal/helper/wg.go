package helper

import "sync"

type WaitGroup struct {
	count       map[string]int
	done        chan struct{}
	lock        sync.Mutex
	DoneFn      func(string)
	doneFnWg    sync.WaitGroup
	DefaultName string
}

func NewWaitGroup() *WaitGroup {
	return &WaitGroup{
		count:       make(map[string]int),
		done:        make(chan struct{}, 1),
		DefaultName: "go",
	}
}

func (wg *WaitGroup) Go(f func()) {
	wg.GoName(wg.DefaultName, f)
}

func (wg *WaitGroup) GoName(name string, f func()) {
	wg.AddName(name, 1)
	go func() {
		f()
		wg.DoneName(name)
	}()
}

func (wg *WaitGroup) Add(n int) {
	wg.AddName(wg.DefaultName, n)
}

func (wg *WaitGroup) AddName(name string, n int) {
	if n < 1 {
		return
	}

	wg.lock.Lock()
	defer wg.lock.Unlock()

	count, _ := wg.count[name]
	wg.count[name] = count + n
}

func (wg *WaitGroup) Done() {
	wg.DoneName(wg.DefaultName)
}

func (wg *WaitGroup) DoneName(name string) {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	n, ok := wg.count[name]
	if !ok {
		return
	}

	if wg.DoneFn != nil {
		wg.doneFnWg.Add(1)
		go func() {
			defer wg.doneFnWg.Done()
			wg.DoneFn(name)
		}()
	}

	n--
	wg.count[name] = n
	if n > 0 {
		return
	}

	delete(wg.count, name)
	if len(wg.count) == 0 {
		close(wg.done)
		wg.done = make(chan struct{}, 0)
	}
}

func (wg *WaitGroup) Wait() {
	wg.lock.Lock()
	done := wg.done
	wg.lock.Unlock()

	<-done
	wg.doneFnWg.Wait()
}

func (wg *WaitGroup) ActiveNames() []string {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	names := make([]string, 0, len(wg.count))
	for name, count := range wg.count {
		for i := 0; i < count; i++ {
			names = append(names, name)
		}
	}
	return names
}
