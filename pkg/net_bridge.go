package pkg

import (
	"io"
	"log"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg/msg"
)

type Bridge interface {
	Forward()
	Read(msg *msg.WspMessage) bool
	Close()
}

type LanBridge struct {
	in     io.WriteCloser
	out    io.ReadWriteCloser
	closed uint32
	stop   chan bool
}

func NewLanBridge(in io.WriteCloser, out io.ReadWriteCloser) *LanBridge {
	return &LanBridge{
		in:   in,
		out:  out,
		stop: make(chan bool),
	}
}
func (b *LanBridge) Read(message *msg.WspMessage) bool {
	if b.IsClosed() {
		return false
	}
	switch message.Cmd {
	case msg.WspCmd_FORWARD:
		_, err := b.out.Write(message.Data)
		if err != nil {
			b.stop <- true
		}
	case msg.WspCmd_CLOSE:
		b.stop <- false
	}
	return true
}
func (b *LanBridge) IsClosed() bool {
	return atomic.LoadUint32(&b.closed) > 0
}
func (b *LanBridge) Close() {
	if b.IsClosed() {
		return
	}
	b.stop <- false
}
func (b *LanBridge) Forward() {
	go func() {
		_, err := io.Copy(b.in, b.out)
		if err != nil {
			log.Println(err)
		}
		b.stop <- true
	}()
	close := <-b.stop
	atomic.AddUint32(&b.closed, 1)
	if close {
		b.in.Close()
	}
	b.out.Close()
}
