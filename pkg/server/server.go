package server

import (
	"log"
	"net/http"
	"strings"

	"nhooyr.io/websocket"
)

type Config struct {
	Auth string `json:"auth,omitempty"`
	Path string `json:"path,omitempty"`
	Port uint16 `json:"port,omitempty"`
}

func (c *Config) clean() {
	c.Path = strings.TrimPrefix(c.Path, "/")
	c.Path = strings.TrimSpace(c.Path)
}
func NewDefaltWsps(config *Config) http.Handler {
	return NewWspsWithHandler(config, http.NotFoundHandler())
}
func NewWspsWithHandler(config *Config, handler http.Handler) http.Handler {
	config.clean()
	wsps := &Wsps{config: config, hub: &Hub{}, handler: handler}
	return wsps
}

type Wsps struct {
	hub     *Hub
	config  *Config
	handler http.Handler
}

func (s *Wsps) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == s.config.Path {
		s.Serve(rw, r)
		return
	}
	paths := strings.Split(path, "/")
	if len(path) == 0 {
		return
	}
	name := paths[0]
	if val, ok := s.hub.LoadHttp(name); ok {
		router := val.(*Router)
		p := router.NewHttpConn(name)
		p.ServeHTTP(rw, r)
		return
	}
	s.handler.ServeHTTP(rw, r)
}

func (s *Wsps) Serve(w http.ResponseWriter, r *http.Request) {
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
