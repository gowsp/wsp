package server

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
	"nhooyr.io/websocket"
)

func (r *conn) ServeHTTP(channel string, rw http.ResponseWriter, req *http.Request) {
	conf, _ := r.LoadConfig(channel)
	if conf.IsHTTP() {
		r.ServeHTTPProxy(conf, rw, req)
	} else {
		r.ServeNetProxy(conf, rw, req)
	}
}
func (r *conn) ServeNetProxy(conf *msg.WspConfig, w http.ResponseWriter, req *http.Request) {
	id := ksuid.New().String()
	signal := make(chan struct{})
	r.channel.NewSession(id, conf, &wsLinker{req, w, signal}, nil).Syn()
	<-signal
}
func (r *conn) ServeHTTPProxy(conf *msg.WspConfig, w http.ResponseWriter, req *http.Request) {
	proxy := r.NewHTTPProxy(conf)
	if proxy == nil {
		return
	}
	proxy.ServeHTTP(w, req)
}
func (r *conn) NewHTTPProxy(conf *msg.WspConfig) *httputil.ReverseProxy {
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
			prefix := "/" + path
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			prepare(r)
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		id := ksuid.New().String()
		writer := r.channel.NewWriter(id)
		conn := NewProxyConn(writer)
		signal := make(chan error)
		err := r.channel.NewSession(id, conf, &httpLinker{conn, signal}, conn).Syn()
		if err != nil {
			return nil, err
		}
		err = <-signal
		return conn, err
	}
	p.Transport = transport
	r.http.Store(channel, p)
	return p
}

type wsLinker struct {
	req    *http.Request
	w      http.ResponseWriter
	signal chan struct{}
}

func (l *wsLinker) InActive(err error) {
	http.Error(l.w, err.Error(), 400)
	l.signal <- struct{}{}
}

func (l *wsLinker) Active(session *channel.Session) error {
	ws, err := websocket.Accept(l.w, l.req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	l.signal <- struct{}{}
	if err != nil {
		log.Printf("websocket accept %v", err)
		return err
	}
	conn := websocket.NetConn(context.Background(), ws, websocket.MessageBinary)
	session.CopyFrom(conn)
	return nil
}

type httpLinker struct {
	conn   net.Conn
	signal chan error
}

func (l *httpLinker) InActive(err error) {
	l.conn.Close()
	l.signal <- err
}

func (l *httpLinker) Active(session *channel.Session) error {
	l.signal <- nil
	return nil
}

func NewProxyConn(writer io.Writer) *ProxyConn {
	r, w := io.Pipe()
	return &ProxyConn{ws: writer, input: w, output: r}
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
	ws     io.Writer
	input  io.WriteCloser
	output io.ReadCloser
}

func (c *ProxyConn) Transport(data *msg.Data) error {
	_, err := c.input.Write(data.Payload())
	return err
}
func (c *ProxyConn) Read(b []byte) (n int, err error) {
	return c.output.Read(b)
}
func (c *ProxyConn) Write(b []byte) (n int, err error) {
	return c.ws.Write(b)
}
func (c *ProxyConn) Close() error {
	return c.input.Close()
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
