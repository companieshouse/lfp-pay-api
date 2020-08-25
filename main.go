package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/dao"
	"github.com/companieshouse/lfp-pay-api/handlers"
	"github.com/gorilla/mux"
)

func main() {
	namespace := "lfp-pay-api"
	log.Namespace = namespace

	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error configuring service: %s. Exiting", err), nil)
		return
	}

	// Create router
	mainRouter := mux.NewRouter()
	svc := dao.NewDAOService(cfg)

	handlers.Register(mainRouter, cfg, svc)

	log.Info("Starting " + namespace)

	h := &http.Server{
		Addr:    cfg.BindAddr,
		Handler: mainRouter,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// run server in new go routine to allow app shutdown signal wait below
	go func() {
		log.Info("starting server...", log.Data{"port": cfg.BindAddr})
		err = h.ListenAndServe()
		log.Info("server stopping...")
		if err != nil && err != http.ErrServerClosed {
			log.Error(err)
			svc.Shutdown()
			os.Exit(1)
		}
	}()

	// wait for app shutdown message before attempting to close server gracefully
	<-stop

	log.Info("shutting down server...")
	svc.Shutdown()
	timeout := time.Duration(5) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = h.Shutdown(ctx)
	if err != nil {
		log.Error(fmt.Errorf("failed to shutdown server gracefully: [%v]", err))
	} else {
		log.Info("server shutdown gracefully")
	}
}
