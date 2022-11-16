package stream

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
)

type Lan interface {
	Rewrite(data *msg.Data)
	Interrupt() error
	io.Closer
}

func newLan(laddr net.Addr, writer *link) net.Conn {
	pr, pw := io.Pipe()
	conn := &conn{
		pr:     pr,
		pw:     pw,
		laddr:  laddr,
		wirter: writer,
		msgs:   make(chan *msg.Data, 64),
	}
	conn.start()
	return conn
}

type conn struct {
	num uint64
	pr  *io.PipeReader
	pw  *io.PipeWriter

	msgs   chan *msg.Data
	laddr  net.Addr
	close  sync.Once
	wirter *link
}

func (c *conn) start() {
	go func() {
		for msg := range c.msgs {
			n, err := c.pw.Write(msg.Payload())
			atomic.AddUint64(&c.num, uint64(n))
			if err != nil {
				c.Close()
			}
		}
		c.pw.Close()
	}()
}
func (c *conn) Rewrite(data *msg.Data) {
	c.msgs <- data
}
func (c *conn) Read(b []byte) (n int, err error) {
	return c.pr.Read(b)
}
func (c *conn) Write(b []byte) (n int, err error) {
	return c.wirter.Write(b)
}
func (c *conn) Interrupt() error {
	c.close.Do(func() {
		close(c.msgs)
	})
	return nil
}
func (c *conn) Close() error {
	c.close.Do(func() {
		close(c.msgs)
		c.wirter.Close()
	})
	return nil
}
func (c *conn) LocalAddr() net.Addr {
	return c.laddr
}
func (c *conn) RemoteAddr() net.Addr {
	return c.wirter.config
}
func (c *conn) SetDeadline(t time.Time) error {
	return nil
}
func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}
