package server

import (
	"sync"
)

func (s *Wsps) Exist(channel string) (exist bool) {
	_, exist = s.channel.Load(channel)
	return
}
func (s *Wsps) Store(channel string, r interface{}) {
	s.channel.Store(channel, r)
}
func (s *Wsps) LoadRouter(channel string) (interface{}, bool) {
	return s.channel.Load(channel)
}
func (s *Wsps) Remove(channel string) {
	s.channel.Delete(channel)
}
func (s *Wsps) Delete(router sync.Map) {
	router.Range(func(key, value interface{}) bool {
		s.channel.Delete(key)
		return true
	})
}
