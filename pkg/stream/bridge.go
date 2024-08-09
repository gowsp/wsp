package stream

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
)

type bridge struct {
	id     string
	taskID uint64
	input  *Wan
	output *Wan
	config *msg.WspConfig

	signal *Signal
	close  sync.Once
}

func (c *bridge) connect(data *msg.Data) error {
	c.id = data.ID()
	c.input.waiting.Store(c.id, c)
	defer c.input.waiting.Delete(c.id)
	if err := c.input.write(*data.Raw, time.Second*5); err != nil {
		return err
	}
	return c.signal.Wait(time.Second * 5)
}

func (c *bridge) ready(resp *msg.Data) error {
	err := parseResponse(resp)
	c.signal.Notify(err)
	if err != nil {
		return err
	}
	c.input.connect.Store(c.id, c)
	c.output.connect.Store(c.id, &bridge{
		taskID: nextTaskID(),
		id:     c.id,
		input:  c.output,
		config: c.config,
		output: c.input,
	})
	c.Rewrite(resp)
	return nil
}
func (c *bridge) TaskID() uint64 {
	return c.taskID
}
func (c *bridge) Rewrite(data *msg.Data) (n int, err error) {
	_, err = c.Write(*data.Raw)
	return
}
func (c *bridge) Read(b []byte) (n int, err error) {
	return 0, io.EOF
}
func (c *bridge) Write(b []byte) (n int, err error) {
	n = len(b)
	err = c.output.write(b, time.Second*5)
	return
}
func (c *bridge) Interrupt() error {
	c.close.Do(func() {
		logger.Info("close bridge %s", c.config.Channel())
		c.output.connect.Delete(c.id)
		data, _ := encode(c.id, msg.WspCmd_INTERRUPT, []byte{})
		c.Write(data)
	})
	return nil
}
func (c *bridge) Close() error {
	c.close.Do(func() {
		logger.Info("close bridge %s", c.config.Channel())
		data, _ := encode(c.id, msg.WspCmd_INTERRUPT, []byte{})
		if val, ok := c.input.connect.LoadAndDelete(c.id); ok {
			val.(*bridge).Write(data)
		}
		if val, ok := c.output.connect.LoadAndDelete(c.id); ok {
			val.(*bridge).Write(data)
		}
	})
	return nil
}
func (c *bridge) LocalAddr() net.Addr {
	return c.config
}
func (c *bridge) RemoteAddr() net.Addr {
	return c.config
}
func (c *bridge) SetDeadline(t time.Time) error {
	return nil
}
func (c *bridge) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *bridge) SetWriteDeadline(t time.Time) error {
	return nil
}
