// Package server is core logic shared by wspc wsps
package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gobwas/ws"
	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
)

type Wsps struct {
	config  *Config
	handler http.Handler
}

func New(config *Config) http.Handler {
	return NewWithHandler(config, http.NotFoundHandler())
}
func NewWithHandler(config *Config, handler http.Handler) http.Handler {
	config.clean()
	wsps := &Wsps{config: config, handler: handler}
	return wsps
}

func (s *Wsps) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.Host)
	channel := "http:domain:" + host
	if val, ok := hub.Load(channel); ok {
		val.(*conn).ServeHTTP(channel, rw, r)
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
	if val, ok := hub.Load(channel); ok {
		val.(*conn).ServeHTTP(channel, rw, r)
		return
	}
	s.handler.ServeHTTP(rw, r)
}

func (s *Wsps) ServeProxy(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Auth") != s.config.Auth {
		w.WriteHeader(401)
		w.Write([]byte("token error, access denied!\n"))
		logger.Error("illegal request %s", getRemoteIP(r))
		return
	}
	proto := r.Header.Get("Proto")
	logger.Info("accept %s, proto: %s", getRemoteIP(r), proto)
	if proto, err := msg.ParseVersion(proto); err != nil || proto.Major() != msg.PROTOCOL_VERSION.Major() {
		w.WriteHeader(400)
		fmt.Fprintf(w, "client proto version %s not support, server proto is %s\n", proto, msg.PROTOCOL_VERSION)
		return
	}
	ws, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		logger.Error("websocket accept %v", err)
		return
	}

	router := &conn{wsps: s}
	router.ListenAndServe(ws)
	router.close()
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
