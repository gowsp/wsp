package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
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
	remote, err := r.wan.DialHTTP(conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	ws, err := websocket.Accept(w, req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		logger.Error("websocket accept %v", err)
		return
	}
	local := websocket.NetConn(context.Background(), ws, websocket.MessageBinary)
	go func() {
		io.Copy(local, remote)
		local.Close()
	}()
	go func() {
		io.Copy(remote, local)
		remote.Close()
	}()
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
	logger.Info("start http proxy %s", channel)
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
		return r.wan.DialHTTP(conf)
	}
	p.Transport = transport
	r.http.Store(channel, p)
	return p
}
