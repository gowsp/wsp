package channel

import (
	"errors"
	"io"
	"net"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

type Session struct {
	id      string
	closed  uint32
	config  *msg.WspConfig
	channel *Channel
	linker  Linker
	writer  Writer
}

func (s *Session) CopyFrom(conn net.Conn) {
	if s.writer == nil {
		writer := s.channel.NewWriter(s.id)
		s.writer = NewTcpWriter(conn, writer)
	}
	io.Copy(s, conn)
	s.Close()
}
func (s *Session) Write(p []byte) (n int, err error) {
	return s.writer.Write(p)
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
		s.InActive(err)
	}
	return err
}
func (s *Session) Ack() error {
	s.channel.session.Store(s.id, s)
	if err := s.linker.Active(s); err != nil {
		s.channel.Reply(s.id, msg.WspCode_FAILED, err.Error())
		return s.InActive(err)
	}
	if err := s.channel.Reply(s.id, msg.WspCode_SUCCESS, ""); err != nil {
		return s.InActive(err)
	}
	return nil
}
func (s *Session) SynAck(data *msg.Data) error {
	var response msg.WspResponse
	if err := proto.Unmarshal(data.Payload(), &response); err != nil {
		return err
	}
	if response.Code == msg.WspCode_FAILED {
		return s.InActive(errors.New(response.Data))
	}
	return s.linker.Active(s)
}
func (s *Session) Transport(data *msg.Data) error {
	err := s.writer.Transport(data)
	if err != nil {
		s.Close()
	}
	return err
}
func (s *Session) InActive(err error) error {
	s.linker.InActive(err)
	return s.Interrupt()
}
func (s *Session) Interrupt() error {
	atomic.AddUint32(&s.closed, 1)
	return s.Close()
}
func (s *Session) Close() error {
	if atomic.LoadUint32(&s.closed) == 0 {
		atomic.AddUint32(&s.closed, 1)
		data := encode(s.id, msg.WspCmd_INTERRUPT, []byte{})
		s.channel.Write(data)
	}
	s.channel.session.Delete(s.id)
	if s.writer == nil {
		return nil
	}
	return s.writer.Close()
}
