package server

import (
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
)

type Config struct {
	Auth string `json:"auth,omitempty"`
	Path string `json:"path,omitempty"`
	Port uint16 `json:"port,omitempty"`
}
type Wsps struct {
	Config  *Config
	network sync.Map
}

func (s *Wsps) Serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Auth") != s.Config.Auth {
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
