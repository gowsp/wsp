package channel

import (
	"io"
	"log"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

type Session struct {
	id      string
	closed  uint32
	config  *msg.WspConfig
	channel *Channel
	handler *Handler
}

func (s *Session) NewWriter() io.Writer {
	return &writer{id: s.id, channel: s.channel}
}
func (s *Session) Syn() error {
	addr, err := proto.Marshal(s.config.ToReqeust())
	if err != nil {
		return err
	}
	s.channel.session.Store(s.id, s)
	data := encode(s.id, msg.WspCmd_CONNECT, addr)
	_, err = s.channel.Write(data)
	if err != nil {
		s.InActive()
	}
	return err
}
func (s *Session) Ack() error {
	s.channel.session.Store(s.id, s)
	if err := s.handler.linker.Active(s); err != nil {
		s.response(msg.WspCode_FAILED, err.Error())
		return s.InActive()
	}
	if err := s.response(msg.WspCode_SUCCESS, ""); err != nil {
		return s.InActive()
	}
	return nil
}
func (s *Session) response(code msg.WspCode, message string) (err error) {
	res := msg.WspResponse{Code: code, Data: message}
	response, _ := proto.Marshal(&res)
	data := encode(s.id, msg.WspCmd_RESPOND, response)
	_, err = s.channel.Write(data)
	return err
}
func (s *Session) SynAck(data *msg.Data) error {
	var response msg.WspResponse
	if err := proto.Unmarshal(data.Payload(), &response); err != nil {
		return err
	}
	if response.Code == msg.WspCode_FAILED {
		log.Println(response.Data)
		return s.InActive()
	}
	return s.handler.linker.Active(s)
}
func (s *Session) Transport(data *msg.Data) error {
	err := s.handler.writer.Transport(data)
	if err != nil {
		s.Close()
	}
	return err
}
func (s *Session) InActive() error {
	s.handler.linker.InActive()
	return s.InActive()
}
func (s *Session) Interrupt() error {
	atomic.AddUint32(&s.closed, 1)
	return s.Close()
}
func (s *Session) Close() error {
	s.channel.session.Delete(s.id)
	if atomic.LoadUint32(&s.closed) == 0 {
		atomic.AddUint32(&s.closed, 1)
		data := encode(s.id, msg.WspCmd_INTERRUPT, []byte{})
		s.channel.Write(data)
	}
	return s.handler.writer.Close()
}
