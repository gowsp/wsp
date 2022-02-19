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
	"nhooyr.io/websocket"
)

func (r *Router) ServeHTTP(channel string, rw http.ResponseWriter, req *http.Request) {
	val, ok := r.channel.Load(channel)
	if !ok {
		r.wsps.Remove(channel)
		http.Error(rw, "router not exist", 404)
		return
	}
	conf := val.(*msg.WspConfig)
	if conf.IsHTTP() {
		r.ServeHTTPProxy(conf, rw, req)
	} else {
		r.ServeNetProxy(conf, rw, req)
	}
}
func (r *Router) ServeNetProxy(conf *msg.WspConfig, w http.ResponseWriter, req *http.Request) {
	id := ksuid.New().String()
	response := make(chan *msg.WspResponse)
	r.routing.AddPending(id, func(data *msg.Data, res *msg.WspResponse) {
		response <- res
	})
	if err := r.wan.Dail(id, conf); err != nil {
		http.Error(w, "router dail error", 500)
		r.routing.DeleteConn(id)
		return
	}
	var res *msg.WspResponse
	select {
	case res = <-response:
	case <-time.After(time.Second * 5):
		res = &msg.WspResponse{Code: msg.WspCode_FAILED, Data: "time out"}
	}
	if res.Code == msg.WspCode_FAILED {
		r.routing.Delete(id)
		http.Error(w, "router not avaiable", 400)
		return
	}
	ws, err := websocket.Accept(w, req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		log.Printf("websocket accept %v", err)
		return
	}
	writer := r.wan.NewWriter(id)
	conn := websocket.NetConn(context.Background(), ws, websocket.MessageBinary)
	repeater := proxy.NewNetRepeater(writer, conn)
	r.routing.AddRepeater(id, repeater)
	repeater.Copy()
}
func (r *Router) ServeHTTPProxy(conf *msg.WspConfig, w http.ResponseWriter, req *http.Request) {
	proxy := r.NewHTTPProxy(conf)
	if proxy == nil {
		return
	}
	proxy.ServeHTTP(w, req)
}
func (r *Router) NewHTTPProxy(conf *msg.WspConfig) *httputil.ReverseProxy {
	channel := conf.Channel()
	c, ok := r.http.Load(channel)
	if ok {
		return c.(*httputil.ReverseProxy)
	}
	log.Printf("start http proxy %s\n", channel)
	u := conf.ReverseURL()
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
			writer := r.wan.NewWriter(id)
			conn := NewProxyConn(writer)
			r.routing.AddRepeater(id, conn.(proxy.Repeater))
			err := r.wan.Dail(id, conf)
			if err != nil {
				r.routing.Delete(id)
				return nil, err
			}
			response := make(chan *msg.WspResponse)
			r.routing.AddPending(id, func(data *msg.Data, res *msg.WspResponse) {
				response <- res
			})
			var res *msg.WspResponse
			select {
			case res = <-response:
			case <-time.After(time.Second * 5):
				res = &msg.WspResponse{Code: msg.WspCode_FAILED, Data: "time out"}
			}
			if res.Code == msg.WspCode_FAILED {
				r.routing.Delete(id)
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
	r.http.Store(channel, p)
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
