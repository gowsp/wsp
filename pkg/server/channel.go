package server

import (
	"sync"
)

var hub = &Hub{}

type Hub struct {
	listen sync.Map
}

func (s *Hub) Load(channel string) (interface{}, bool) {
	return s.listen.Load(channel)
}
func (s *Hub) Exist(channel string) (exist bool) {
	_, exist = s.listen.Load(channel)
	return
}
func (s *Hub) Store(channel string, r interface{}) {
	s.listen.Store(channel, r)
}
func (s *Hub) Remove(channel string) {
	s.listen.Delete(channel)
}
