package pkg

import (
	"errors"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
)

var ErrConnNotExist = errors.New("connection does not exist")

type Pending struct {
	OnReponse func(*msg.Data)
}

func NewRouting() *Routing {
	return &Routing{}
}

type Routing struct {
	tables  sync.Map
	pending sync.Map
}

func (s *Routing) AddPending(id string, repeater *Pending) {
	s.pending.Store(id, repeater)
}
func (s *Routing) AddRepeater(id string, repeater Repeater) {
	s.tables.Store(id, repeater)
}
func (s *Routing) Delete(id string) {
	s.tables.Delete(id)
	s.pending.Delete(id)
}
func (s *Routing) Routing(data *msg.Data) error {
	switch data.Msg.Cmd {
	case msg.WspCmd_CONN_REP:
		if val, ok := s.pending.Load(data.Id()); ok {
			s.pending.Delete(data.Id())
			go val.(*Pending).OnReponse(data)
			return nil
		}
		return ErrConnNotExist
	case msg.WspCmd_FORWARD:
		if val, ok := s.tables.Load(data.Id()); ok {
			val.(Repeater).Relay(data)
			return nil
		}
		return ErrConnNotExist
	case msg.WspCmd_CLOSE:
		if val, ok := s.tables.Load(data.Id()); ok {
			s.tables.Delete(data.Id())
			val.(Repeater).Interrupt()
		}
	}
	return nil
}
