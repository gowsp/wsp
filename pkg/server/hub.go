package server

import (
	"sync"
)

type Hub struct {
	http  sync.Map
	local sync.Map
}

func (b *Hub) AddLocal(key string, r interface{}) {
	b.local.Store(key, r)
}
func (b *Hub) AddHttp(key string, r interface{}) {
	b.http.Store(key, r)
}
func (b *Hub) ExistHttp(key string) bool {
	_, ok := b.http.Load(key)
	return ok
}
func (b *Hub) ExistLocal(key string) bool {
	_, ok := b.local.Load(key)
	return ok
}
func (b *Hub) LoadLocal(addr string) (interface{}, bool) {
	return b.local.Load(addr)
}
func (b *Hub) LoadHttp(addr string) (interface{}, bool) {
	return b.http.Load(addr)
}
func (b *Hub) Delete(router *Hub) {
	router.http.Range(func(key, value interface{}) bool {
		b.http.Delete(key)
		return true
	})
	router.local.Range(func(key, value interface{}) bool {
		b.local.Delete(key)
		return true
	})
}
