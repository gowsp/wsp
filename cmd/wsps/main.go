package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gowsp/wsp/pkg/server"
)

func main() {
	config, err := parseConf()
	if err != nil {
		log.Println(err)
		return
	}
	wsps := server.NewDefaltWsps(config)
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

func parseConf() (*server.Config, error) {
	configVar := flag.String("c", "wsps.json", "wsps config file ")
	flag.Parse()
	file, err := os.Open(*configVar)
	if err != nil {
		return nil, err
	}
	conf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var config server.Config
	err = json.Unmarshal(conf, &config)
	return &config, err
}
