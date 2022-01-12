package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

func (r *Router) AddHttpConn(id string, addr *msg.WspAddr) {
	path := r.GlobalConfig().Path
	if path == addr.Address {
		r.wan.CloseRemote(id, fmt.Sprintf("name %s not allowd", addr))
		return
	}
	if r.GlobalHub().ExistLocal(addr.Address) {
		r.wan.CloseRemote(id, fmt.Sprintf("name %s registered", addr))
	} else {
		log.Printf("server register address %s", addr.Address)
		r.hub.AddHttp(addr.Address, addr)
		r.GlobalHub().AddHttp(addr.Address, r)
	}
}

func (router *Router) NewHttpConn(name string) *httputil.ReverseProxy {
	val, ok := router.hub.LoadHttp(name)
	if !ok {
		return nil
	}
	c, ok := router.http.Load(name)
	if ok {
		return c.(*httputil.ReverseProxy)
	}
	addr := val.(*msg.WspAddr)
	log.Printf("start http proxy %s\n", name)
	target, _ := url.Parse(addr.Domain)
	p := httputil.NewSingleHostReverseProxy(target)
	prepare := p.Director
	p.Director = func(r *http.Request) {
		prepare(r)
		prefix := "/" + name
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
	}
	p.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			id := ksuid.New().String()
			writer := router.wan.NewWriter(id)
			conn := NewProxyConn(writer)
			router.routing.AddRepeater(id, conn.(pkg.Repeater))
			err := router.wan.Dail(id, msg.WspType_HTTP, name)
			if err != nil {
				router.routing.Delete(id)
				return nil, err
			}
			response := make(chan byte)
			router.routing.AddPending(id, &pkg.Pending{OnReponse: func(message *msg.Data) {
				response <- message.Payload()[0]
			}})
			var res byte
			select {
			case res = <-response:
			case <-time.After(time.Second * 5):
				res = 0
			}
			if res == 0 {
				router.routing.Delete(id)
				return nil, errors.New("http connect error")
			}
			return conn, nil
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	router.http.Store(name, p)
	return p
}

func NewProxyConn(writer io.WriteCloser) net.Conn {
	buff := pkg.BufPool.Get().(*bytes.Buffer)
	return &ProxyConn{input: buff, output: writer, sign: make(chan struct{}, 8)}
}

type wpsAddr struct {
}

func (a wpsAddr) Network() string {
	return "wsp"
}

func (a wpsAddr) String() string {
	return "wsp/unknown-addr"
}

type ProxyConn struct {
	input  *bytes.Buffer
	output io.WriteCloser
	closed uint32
	sign   chan struct{}
	rw     sync.RWMutex
}

func (c *ProxyConn) Relay(data *msg.Data) error {
	c.rw.Lock()
	defer c.rw.Unlock()

	_, err := c.input.Write(data.Payload())
	c.sign <- struct{}{}
	return err
}
func (c *ProxyConn) Interrupt() error {
	return c.Close()
}

func (c *ProxyConn) Read(b []byte) (n int, err error) {
	if atomic.LoadUint32(&c.closed) > 0 {
		return 0, io.EOF
	}
	c.rw.RLock()
	n, err = c.input.Read(b)
	c.rw.RUnlock()
	if n > 0 {
		return
	}
	select {
	case <-c.sign:
		c.rw.RLock()
		n, err = c.input.Read(b)
		c.rw.RUnlock()
	case <-time.After(time.Second * 10):
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func (c *ProxyConn) Write(b []byte) (n int, err error) {
	return c.output.Write(b)
}

func (c *ProxyConn) Close() error {
	if atomic.LoadUint32(&c.closed) > 0 {
		return nil
	}
	atomic.AddUint32(&c.closed, 1)
	c.input.Reset()
	pkg.BufPool.Put(c.input)
	return c.output.Close()
}

func (c *ProxyConn) LocalAddr() net.Addr {
	return wpsAddr{}
}

func (c *ProxyConn) RemoteAddr() net.Addr {
	return wpsAddr{}
}

func (c *ProxyConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *ProxyConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *ProxyConn) SetWriteDeadline(t time.Time) error {
	return nil
}
