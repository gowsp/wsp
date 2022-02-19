package wsp

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/gowsp/wsp/pkg/client"
	"github.com/gowsp/wsp/pkg/server"
)

func TestShutdown(t *testing.T) {

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log.Println(ctx)

	log.Println("Server exiting")
}

func TestServer(t *testing.T) {
	wsps := server.NewWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	http.ListenAndServe(":8080", wsps)
}
func TestClient(t *testing.T) {
	client := client.Wspc{Config: &client.Config{
		Auth:    "auth",
		Server:  "ws://127.0.0.1:8080/proxy",
		Dynamic: []string{"socks5://ssh:passwod@127.0.0.1:1088"},
		Remote: []string{
			"tcp://ssh:ssh@192.168.1.200:22",
			"tcp://rdp:rdp@192.168.1.200:3389",
		},
	}}
	client.ListenAndServe()
}
func TestSocks5Proxy(t *testing.T) {
	wsps := server.NewWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	client := client.NewWspc(&client.Config{
		Auth:    "auth",
		Server:  "ws://127.0.0.1:8080/proxy",
		Dynamic: []string{"socks5://:1088"},
	})
	client.ListenAndServe()
	<-c
	log.Println("closed")
}

func TestHTTPProxy(t *testing.T) {
	wsps := server.NewWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	client := client.Wspc{Config: &client.Config{
		Auth:    "auth",
		Server:  "ws://127.0.0.1:8080/proxy",
		Dynamic: []string{"http://:1088"},
	}}
	client.ListenAndServe()
	<-c
	log.Println("closed")
}

func TestTCPOverHTTPProxy(t *testing.T) {
	wsps := server.NewWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	client := client.Wspc{Config: &client.Config{
		Auth:   "auth",
		Server: "ws://127.0.0.1:8080/proxy",
		Remote: []string{"tcp://10.0.0.4:5900?mode=path&value=vnc"},
	}}
	client.ListenAndServe()
	<-c
	log.Println("closed")
}

func TestProxy(t *testing.T) {
	wsps := server.NewWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	sshConfig := &client.Config{
		Auth:   "auth",
		Server: "ws://127.0.0.1:8080/proxy",
		Remote: []string{"tcp://ssh:ssh@10.0.0.2:22"},
	}
	l := client.NewWspc(sshConfig)
	go l.ListenAndServe()
	time.Sleep(1 * time.Second)
	v := client.NewWspc(sshConfig)
	go v.ListenAndServe()
	time.Sleep(1 * time.Second)

	config := &client.Config{
		Auth:   "auth",
		Server: "ws://127.0.0.1:8080/proxy",
		Local:  []string{"tcp://ssh:ssh@127.0.0.1:2200"},
	}
	r := client.NewWspc(config)
	go r.ListenAndServe()
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	<-s
}

func TestHttp(t *testing.T) {
	wsps := server.NewWsps(&server.Config{Auth: "auth"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	server := http.NewServeMux()
	server.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		msg := "index"
		rw.Write([]byte(msg))
	})
	server.HandleFunc("/api", func(rw http.ResponseWriter, r *http.Request) {
		msg := "api"
		rw.Write([]byte(msg))
	})
	web := http.Server{Handler: server, Addr: ":8010"}
	go web.ListenAndServe()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	client := client.Wspc{Config: &client.Config{Auth: "auth",
		Server: "ws://127.0.0.1:8080",
		Remote: []string{"http://127.0.0.1:8010?mode=path&value=api"},
	}}
	client.ListenAndServe()
	<-c
	log.Println("closed")
}
