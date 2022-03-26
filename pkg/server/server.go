// Package server is core logic shared by wspc wsps
package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"nhooyr.io/websocket"
)

func NewWsps(config *Config) http.Handler {
	return NewWspsWithHandler(config, http.NotFoundHandler())
}
func NewWspsWithHandler(config *Config, handler http.Handler) http.Handler {
	config.clean()
	wsps := &Wsps{config: config, handler: handler}
	return wsps
}

type Wsps struct {
	config  *Config
	channel sync.Map
	handler http.Handler
}

func (s *Wsps) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.Host)
	channel := "http:domain:" + host
	if val, ok := s.LoadRouter(channel); ok {
		router := val.(*Router)
		router.ServeHTTP(channel, rw, r)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == s.config.Path {
		s.ServeProxy(rw, r)
		return
	}
	paths := strings.Split(path, "/")
	if len(path) == 0 {
		s.handler.ServeHTTP(rw, r)
		return
	}
	channel = "http:path:" + paths[0]
	if val, ok := s.LoadRouter(channel); ok {
		router := val.(*Router)
		router.ServeHTTP(channel, rw, r)
		return
	}
	s.handler.ServeHTTP(rw, r)
}

func (s *Wsps) ServeProxy(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Auth") != s.config.Auth {
		w.WriteHeader(401)
		w.Write([]byte("token error, access denied!\n"))
		return
	}
	proto := r.Header.Get("Proto")
	log.Printf("accept %s, proto: %s", getRemoteIP(r), proto)
	if proto, err := msg.ParseVersion(proto); err != nil || proto.Major() != msg.PROTOCOL_VERSION.Major() {
		w.WriteHeader(400)
		fmt.Fprintf(w, "client proto version %s not support, server proto is %s\n", proto, msg.PROTOCOL_VERSION)
		return
	}
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		log.Printf("websocket accept %v", err)
		return
	}
	defer ws.Close(websocket.StatusNormalClosure, "close connect")

	router := &Router{wsps: s, routing: proxy.NewRouting()}
	router.wan = proxy.NewWan(ws, router)
	router.wan.Serve()
}

func getRemoteIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}
