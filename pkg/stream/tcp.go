package stream

import (
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg/msg"
)

func newTCP(local net.Conn, writer io.WriteCloser) io.WriteCloser {
	conn := &tcpConn{
		local:  local,
		remote: writer,
		msgs:   make(chan *msg.Data, 64),
	}
	conn.start()
	return conn
}

type tcpConn struct {
	num    uint64
	remote io.WriteCloser
	local  net.Conn
	msgs   chan *msg.Data
	close  sync.Once
}

func (c *tcpConn) start() {
	go func() {
		for msg := range c.msgs {
			n, err := c.local.Write(msg.Payload())
			atomic.AddUint64(&c.num, uint64(n))
			if err != nil {
				c.Close()
			}
		}
		c.local.Close()
	}()
}
func (c *tcpConn) Rewrite(data *msg.Data) {
	c.msgs <- data
}
func (c *tcpConn) Write(b []byte) (n int, err error) {
	return c.remote.Write(b)
}
func (c *tcpConn) Interrupt() error {
	c.close.Do(func() {
		close(c.msgs)
	})
	return nil
}
func (c *tcpConn) Close() error {
	c.close.Do(func() {
		close(c.msgs)
		c.remote.Close()
	})
	return nil
}
