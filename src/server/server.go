package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"vaultlink/vault"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	server *http.Server
	vault  *vault.Vault
}

func New(vault *vault.Vault, port int) *Server {
	srv := &Server{vault: vault, server: &http.Server{Addr: fmt.Sprintf(":%v", port)}}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.Serve)
	srv.server.Handler = mux
	return srv
}

func (srv *Server) Listen() {
	go func() {
		if err := srv.server.ListenAndServe(); err != nil {
			log.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	log.Info("Server started")

	// listening OS shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	srv.server.Shutdown(context.Background())
}

func (srv *Server) Serve(w http.ResponseWriter, r *http.Request) {
	if err := srv.vault.Ping(); err != nil {
		log.Errorf("vault ping error:%s", err)
		http.Error(w, fmt.Sprintf("vault ping error: %v", err), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "ok")
}
