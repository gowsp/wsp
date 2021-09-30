package client

import (
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) ListenRemote(addr Addr) {
	local, err := net.Listen("tcp", addr.Address())
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := local.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		c.NewRemoteConn(conn, addr.Name, addr.Secret)
	}
}

func (c *Wspc) NewRemoteConn(out net.Conn, address, secret string) {
	log.Println("open remote connect", address)
	id := ksuid.New().String()

	defer out.Close()

	in := c.wan.NewWriter(id)
	bridge := NewLanBridge(in, out)
	c.lan.Store(id, bridge)
	defer c.lan.Delete(id)

	err := c.wan.SecretDail(id, msg.WspType_REMOTE, address, secret)
	if err != nil {
		log.Println(err)
		return
	}

	bridge.Forward()
	log.Println("close remote connect", address)
}

type LanBridge struct {
	in      io.WriteCloser
	out     io.ReadWriteCloser
	closed  uint32
	closeIn chan bool
}

func NewLanBridge(in io.WriteCloser, out io.ReadWriteCloser) pkg.Bridge {
	return &LanBridge{
		in:      in,
		out:     out,
		closeIn: make(chan bool),
	}
}
func (b *LanBridge) Read(message *msg.WspMessage) bool {
	if b.IsClosed() {
		return false
	}
	switch message.Cmd {
	case msg.WspCmd_CONN_REP:
		if message.Data[0] == 0 {
			b.closeIn <- false
			return true
		}
		log.Println("remote connection succeeded")
		go func() {
			_, err := io.Copy(b.in, b.out)
			if err != nil {
				log.Println(err)
			}
			b.closeIn <- true
		}()
	case msg.WspCmd_FORWARD:
		_, err := b.out.Write(message.Data)
		if err != nil {
			b.closeIn <- true
		}
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
	b.closeIn <- false
}
func (b *LanBridge) Forward() {
	close := <-b.closeIn
	atomic.AddUint32(&b.closed, 1)
	if close {
		b.in.Close()
	}
}
