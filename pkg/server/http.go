package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"github.com/segmentio/ksuid"
)

func (router *Router) ServeHTTP(channel string, rw http.ResponseWriter, r *http.Request) {
	proxy := router.NewHttpProxy(channel)
	if proxy == nil {
		http.Error(rw, "router not exist", 404)
		return
	}
	proxy.ServeHTTP(rw, r)
}
func (router *Router) NewHttpProxy(channel string) *httputil.ReverseProxy {
	val, ok := router.channel.Load(channel)
	if !ok {
		router.wsps.Remove(channel)
		return nil
	}
	c, ok := router.http.Load(channel)
	if ok {
		return c.(*httputil.ReverseProxy)
	}
	conf := val.(*msg.WspConfig)
	log.Printf("start http proxy %s\n", channel)
	u := conf.ReverseUrl()
	p := httputil.NewSingleHostReverseProxy(u)
	prefix := "http:path:"
	if strings.HasPrefix(channel, prefix) {
		prepare := p.Director
		path := strings.TrimPrefix(channel, prefix)
		p.Director = func(r *http.Request) {
			prepare(r)
			prefix := "/" + path
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}
	}
	p.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			id := ksuid.New().String()
			writer := router.wan.NewWriter(id)
			conn := NewProxyConn(writer)
			router.routing.AddRepeater(id, conn.(proxy.Repeater))
			err := router.wan.Dail(id, conf)
			if err != nil {
				router.routing.Delete(id)
				return nil, err
			}
			response := make(chan *msg.WspResponse)
			router.routing.AddPending(id, &proxy.Pending{OnReponse: func(data *msg.Data, res *msg.WspResponse) {
				response <- res
			}})
			var res *msg.WspResponse
			select {
			case res = <-response:
			case <-time.After(time.Second * 5):
				res = &msg.WspResponse{Code: msg.WspCode_FAILED, Data: "time out"}
			}
			if res.Code == msg.WspCode_FAILED {
				router.routing.Delete(id)
				return nil, errors.New(res.Data)
			}
			return conn, nil
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	router.http.Store(channel, p)
	return p
}

func NewProxyConn(writer io.WriteCloser) net.Conn {
	buff := proxy.GetBuffer()
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
	proxy.PutBuffer(c.input)
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
