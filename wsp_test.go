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

func TestSocks5(t *testing.T) {
	wsps := server.NewDefaltWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	client := client.Wspc{Config: &client.Config{Auth: "auth", Server: "ws://127.0.0.1:8080/proxy", Socks5: ":1088"}}
	client.ListenAndServe()
	<-c
	log.Println("closed")
}

func TestProxy(t *testing.T) {
	wsps := server.NewDefaltWsps(&server.Config{Auth: "auth", Path: "/proxy"})
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	config := &client.Config{Auth: "auth",
		Server: "ws://127.0.0.1:8080/proxy",
		Socks5: ":1080",
		Addrs:  []client.Addr{{Forward: "local", Name: "demo", LocalAddr: "192.168.5.16", LocalPort: 22}},
	}
	l := client.Wspc{Config: config}
	go l.ListenAndServe()
	time.Sleep(1 * time.Second)

	config = &client.Config{
		Auth:   "auth",
		Server: "ws://127.0.0.1:8080/proxy",
		Addrs:  []client.Addr{{Forward: "remote", Name: "demo", LocalAddr: "127.0.0.1", LocalPort: 9909}},
	}
	r := client.Wspc{Config: config}
	go r.ListenAndServe()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	<-s
}

func TestHttp(t *testing.T) {
	config := &server.Config{Auth: "auth"}
	wsps := server.NewDefaltWsps(config)
	go http.ListenAndServe(":8080", wsps)
	time.Sleep(1 * time.Second)

	server := http.NewServeMux()
	server.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		msg := "index"
		rw.Write([]byte(msg))
	})
	server.HandleFunc("/greet", func(rw http.ResponseWriter, r *http.Request) {
		msg := "hello"
		rw.Write([]byte(msg))
	})
	web := http.Server{Handler: server, Addr: ":8010"}
	go web.ListenAndServe()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	client := client.Wspc{Config: &client.Config{Auth: "auth",
		Server: "ws://127.0.0.1:8080",
		Addrs:  []client.Addr{{Forward: "http", Name: "proxy", LocalAddr: "127.0.0.1", LocalPort: 8010}},
	}}
	client.ListenAndServe()
	<-c
	log.Println("closed")
}
