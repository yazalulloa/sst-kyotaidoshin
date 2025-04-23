package api

import (
	"sync"
	"time"
)

type item struct {
	value      string
	lastAccess int64
}

type TTLMap struct {
	m map[string]*item
	l sync.Mutex
}

func New(ln int, maxTTL time.Duration) (m *TTLMap) {
	m = &TTLMap{m: make(map[string]*item, ln)}
	go func() {
		for now := range time.Tick(time.Second) {
			m.l.Lock()
			for k, v := range m.m {
				if now.UnixMilli()-v.lastAccess > maxTTL.Milliseconds() {
					delete(m.m, k)
				}
			}
			m.l.Unlock()
		}
	}()
	return
}

func (m *TTLMap) Len() int {
	return len(m.m)
}

func (m *TTLMap) Put(k, v string) {
	m.l.Lock()
	it, ok := m.m[k]
	if !ok {
		it = &item{value: v}
		m.m[k] = it
	}
	it.lastAccess = time.Now().UnixMilli()
	m.l.Unlock()
}

func (m *TTLMap) Get(k string) (v string) {
	m.l.Lock()
	if it, ok := m.m[k]; ok {
		v = it.value
		it.lastAccess = time.Now().UnixMilli()
	}
	m.l.Unlock()
	return

}

func (m *TTLMap) Delete(k string) {
	m.l.Lock()
	delete(m.m, k)
	m.l.Unlock()
}

func (m *TTLMap) Clear() {
	m.l.Lock()
	m.m = make(map[string]*item)
	m.l.Unlock()
}
