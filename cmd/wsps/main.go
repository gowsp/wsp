package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gowsp/wsp/internal/cmd"
	"github.com/gowsp/wsp/pkg/server"
)

var date string
var version string

func main() {
	config := &server.Config{}
	err := cmd.ParseConfig(config, cmd.NewVersion(date, version))
	if err != nil {
		log.Println(err)
		return
	}
	wsps := server.New(config)
	addr := fmt.Sprintf(":%d", config.Port)
	srv := &http.Server{Handler: wsps, Addr: addr}
	go func() {
		log.Printf("wsps will start at %s, path %s ", addr, config.Path)
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("wsps forced to shutdown:", err)
	}
	log.Println("wsps exiting")
}
