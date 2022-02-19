package proxy

import (
	"errors"
	"log"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

var ErrConnNotExist = errors.New("connection does not exist")

type Pending func(*msg.Data, *msg.WspResponse)

func NewRouting() *Routing {
	return &Routing{}
}

type Routing struct {
	connect sync.Map
	accept  sync.Map
}

func (s *Routing) AddPending(id string, repeater Pending) {
	s.connect.Store(id, repeater)
}
func (s *Routing) DeleteConn(id string) {
	s.connect.Delete(id)
}
func (s *Routing) AddRepeater(id string, repeater Repeater) {
	s.accept.Store(id, repeater)
}
func (s *Routing) Delete(id string) {
	s.accept.Delete(id)
	s.connect.Delete(id)
}
func (s *Routing) Routing(data *msg.Data) error {
	id := data.ID()
	switch data.Msg.Cmd {
	case msg.WspCmd_RESPOND:
		if val, ok := s.connect.Load(id); ok {
			s.connect.Delete(id)
			var response msg.WspResponse
			proto.Unmarshal(data.Payload(), &response)
			go val.(Pending)(data, &response)
			return nil
		}
		return ErrConnNotExist
	case msg.WspCmd_TRANSFER:
		if val, ok := s.accept.Load(id); ok {
			val.(Repeater).Relay(data)
			return nil
		}
		return ErrConnNotExist
	case msg.WspCmd_INTERRUPT:
		if val, ok := s.accept.Load(id); ok {
			s.accept.Delete(data.ID())
			val.(Repeater).Interrupt()
		}
	default:
		log.Println("unknown command")
	}
	return nil
}
func (s *Routing) Close() error {
	s.connect.Range(func(key, value interface{}) bool {
		value.(Repeater).Close()
		return true
	})
	return nil
}
