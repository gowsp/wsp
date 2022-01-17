package server

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"nhooyr.io/websocket"
)

type Config struct {
	Host string `json:"host,omitempty"`
	Auth string `json:"auth,omitempty"`
	Path string `json:"path,omitempty"`
	Port uint16 `json:"port,omitempty"`
}

func (c *Config) clean() {
	c.Path = strings.TrimPrefix(c.Path, "/")
	c.Path = strings.TrimSpace(c.Path)
}
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
		w.Write([]byte("Access denied!\n"))
		return
	}
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		log.Printf("websocket accept %v", err)
		return
	}
	defer ws.Close(websocket.StatusNormalClosure, "close connect")

	router := s.NewRouter(ws)
	router.ServeConn()
}
