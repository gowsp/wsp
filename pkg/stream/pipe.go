package stream

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
)

type Conn interface {
	TaskID() uint64
	Rewrite(data *msg.Data) (n int, err error)
	Interrupt() error
	io.Closer
}

func newConn(local net.Addr, remote *link) *conn {
	pr, pw := io.Pipe()
	return &conn{
		taskID: nextTaskID(),
		pr:     pr,
		pw:     pw,
		local:  local,
		remote: remote,
	}
}

type conn struct {
	taskID uint64
	local  net.Addr
	remote *link
	pr     *io.PipeReader
	pw     *io.PipeWriter
	close  sync.Once
}

func (c *conn) TaskID() uint64 {
	return c.taskID
}
func (c *conn) Rewrite(data *msg.Data) (n int, err error) {
	return c.pw.Write(data.Payload())
}
func (c *conn) Read(p []byte) (n int, err error) {
	return c.pr.Read(p)
}
func (c *conn) Write(p []byte) (n int, err error) {
	return c.remote.Write(p)
}
func (c *conn) Interrupt() error {
	return c.pw.Close()
}
func (c *conn) Close() error {
	c.close.Do(func() {
		c.pw.Close()
		c.remote.Close()
	})
	return nil
}
func (c *conn) LocalAddr() net.Addr                { return c.local }
func (c *conn) RemoteAddr() net.Addr               { return c.remote.config }
func (c *conn) SetDeadline(t time.Time) error      { return nil }
func (c *conn) SetReadDeadline(t time.Time) error  { return nil }
func (c *conn) SetWriteDeadline(t time.Time) error { return nil }
