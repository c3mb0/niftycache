package niftycache

import (
	"sync"
	"time"

	"github.com/eapache/queue"
)

type Cache struct {
	ttl          time.Duration
	removeCB     callback
	setCB        callback
	updateCB     callback
	expireCB     callback
	extendTTL    bool
	items        map[string]*item
	ih           *itemsHeap
	m            *sync.Mutex
	maxExpires   int
	maxCallbacks int
	cbLimiter    chan struct{}
	done         chan struct{}
	wg           *sync.WaitGroup
	closed       bool
	callbacks    *queue.Queue
}

type Option func(*Cache)

type Callback func(string, interface{})

type callback func(string, interface{}) func()

func createCBClosure(f Callback) callback {
	return func(k string, v interface{}) func() {
		return func() {
			f(k, v)
		}
	}
}

func RemoveCallback(f Callback) Option {
	return func(nc *Cache) {
		nc.removeCB = createCBClosure(f)
	}
}

func UpdateCallback(f Callback) Option {
	return func(nc *Cache) {
		nc.updateCB = createCBClosure(f)
	}
}

func SetCallback(f Callback) Option {
	return func(nc *Cache) {
		nc.setCB = createCBClosure(f)
	}
}

func ExpireCallback(f Callback) Option {
	return func(nc *Cache) {
		nc.expireCB = createCBClosure(f)
	}
}

func ExtendTTLOnHit() Option {
	return func(nc *Cache) {
		nc.extendTTL = true
	}
}

func MaxExpires(maxExpires int) Option {
	return func(nc *Cache) {
		nc.maxExpires = maxExpires
	}
}

func MaxCallbacks(maxCallbacks int) Option {
	return func(nc *Cache) {
		nc.maxCallbacks = maxCallbacks
	}
}

func New(ttl time.Duration, options ...Option) *Cache {
	nc := &Cache{
		ttl:          ttl,
		items:        make(map[string]*item),
		ih:           newItemsHeap(),
		m:            new(sync.Mutex),
		maxExpires:   10000,
		maxCallbacks: 1000,
		callbacks:    queue.New(),
		done:         make(chan struct{}),
		wg:           new(sync.WaitGroup),
	}
	for _, o := range options {
		o(nc)
	}
	nc.cbLimiter = make(chan struct{}, nc.maxCallbacks)
	nc.wg.Add(2)
	go nc.handleExpirations()
	go nc.handleCallbacks()
	return nc
}

func (nc *Cache) Close() {
	close(nc.done)
	nc.wg.Wait()
	nc.m.Lock()
	nc.items = nil
	nc.ih = nil
	nc.callbacks = nil
	nc.closed = true
	nc.m.Unlock()
}

func (nc *Cache) handleCallbacks() {
	done := nc.done
	for {
		select {
		case <-done:
			done = nil
		default:
			nc.m.Lock()
			if nc.callbacks.Length() == 0 {
				nc.m.Unlock()
				if done == nil {
					nc.wg.Done()
					return
				}
				time.Sleep(time.Second)
				continue
			}
			out := nc.callbacks.Remove()
			nc.m.Unlock()
			cb := out.(func())
			nc.cbLimiter <- struct{}{}
			go func() {
				cb()
				<-nc.cbLimiter
			}()
		}
	}

}

func (nc *Cache) handleExpirations() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-nc.done:
			t.Stop()
			nc.wg.Done()
			return
		case <-t.C:
			nc.m.Lock()
			for item, j := nc.ih.peek(), 0; j < nc.maxExpires && item != nil && item.expired(); item, j = nc.ih.peek(), j+1 {
				delete(nc.items, item.key)
				nc.ih.pop()
				if nc.expireCB != nil {
					nc.callbacks.Add(nc.expireCB(item.key, item.value))
				}
			}
			nc.m.Unlock()
		}
	}
}

func (nc *Cache) Remove(key string) {
	nc.m.Lock()
	defer nc.m.Unlock()
	if nc.closed {
		return
	}
	item, ok := nc.items[key]
	if !ok {
		return
	}
	delete(nc.items, key)
	nc.ih.remove(item)
	if nc.removeCB != nil {
		nc.callbacks.Add(nc.removeCB(key, item.value))
	}
}

func (nc *Cache) Get(key string) (interface{}, bool) {
	nc.m.Lock()
	defer nc.m.Unlock()
	if nc.closed {
		return nil, false
	}
	item, ok := nc.items[key]
	if !ok {
		return nil, false
	}
	if nc.extendTTL {
		item.touch()
		nc.ih.update(item)
	}
	return item.value, true
}

func (nc *Cache) Set(key string, value interface{}) {
	nc.m.Lock()
	defer nc.m.Unlock()
	if nc.closed {
		return
	}
	item, ok := nc.items[key]
	if !ok {
		item = newItem(key, value, nc.ttl)
		nc.items[key] = item
		nc.ih.push(item)
		if nc.setCB != nil {
			nc.callbacks.Add(nc.setCB(key, value))
		}
	} else {
		item.update(value)
		nc.ih.update(item)
		if nc.updateCB != nil {
			nc.callbacks.Add(nc.updateCB(key, value))
		}
	}
}
