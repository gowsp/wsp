package stream

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
)

type bridge struct {
	id     string
	input  *Wan
	output *Wan
	config *msg.WspConfig

	num uint64

	msgs   chan *msg.Data
	signal chan struct{}
	start  sync.Once
	close  sync.Once
}

func (c *bridge) connect(data *msg.Data) error {
	c.id = data.ID()
	c.input.waiting.Store(c.id, c)
	defer c.input.waiting.Delete(c.id)
	if err := c.input.write(*data.Raw, time.Second*5); err != nil {
		return err
	}
	select {
	case <-c.signal:
		return nil
	case <-time.After(time.Second * 5):
		return errors.New("timeout")
	}
}

func (c *bridge) ready(resp *msg.Data) error {
	c.signal <- struct{}{}
	if err := parseResponse(resp); err != nil {
		c.Write(*resp.Raw)
		return err
	}
	c.input.connect.Store(c.id, c)
	c.output.connect.Store(c.id, &bridge{
		id:     c.id,
		input:  c.output,
		config: c.config,
		output: c.input,
	})
	c.Rewrite(resp)
	return nil
}
func (c *bridge) run() {
	c.msgs = make(chan *msg.Data, 64)
	go func() {
		for msg := range c.msgs {
			_, err := c.Write(*msg.Raw)
			atomic.AddUint64(&c.num, uint64(len(msg.Payload())))
			if err != nil {
				log.Println("brideg error")
				c.Close()
			}
		}
	}()
}
func (c *bridge) Rewrite(data *msg.Data) {
	c.start.Do(c.run)
	c.msgs <- data
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
		close(c.msgs)
		if val, ok := c.output.connect.LoadAndDelete(c.id); ok {
			close(val.(*bridge).msgs)
		}
		log.Println("close bridge", c.config.Channel())
		data, _ := encode(c.id, msg.WspCmd_INTERRUPT, []byte{})
		c.Write(data)
	})
	return nil
}
func (c *bridge) Close() error {
	c.close.Do(func() {
		log.Println("close bridge", c.config.Channel())
		data, _ := encode(c.id, msg.WspCmd_INTERRUPT, []byte{})
		if val, ok := c.input.connect.LoadAndDelete(c.id); ok {
			close(val.(*bridge).msgs)
			val.(*bridge).Write(data)
		}
		if val, ok := c.output.connect.LoadAndDelete(c.id); ok {
			close(val.(*bridge).msgs)
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
