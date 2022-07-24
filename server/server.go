package server

import (
	"context"
	"embed"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/dmkorzh/history-service/server/handlers"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type HttpServer struct {
	server   http.Server
	listener net.Listener
	ctrl     handlers.Controller
	done     chan struct{}
}

//go:embed sql
var content embed.FS

func NewServer(conn clickhouse.Conn, addr string) (*HttpServer, error) {
	addrToListen, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	listen, err := net.ListenTCP("tcp", addrToListen)
	if err != nil {
		return nil, err
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	serv := &HttpServer{
		server: http.Server{
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		listener: listen,
		ctrl: handlers.Controller{
			Database: &handlers.Database{Driver: conn, Table: "calls"},
		},
		done: make(chan struct{}),
	}
	serv.makeRouter(router)
	serv.ctrl.Templates, err = parseTemplates(router)
	if err != nil {
		return nil, err
	}
	log.Infof("Start listening HTTP on %s", serv.listener.Addr().String())
	return serv, nil
}

func (s *HttpServer) Serve(ctx context.Context) {
	errC := make(chan error)
	go func() {
		if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errC <- err
		}
		close(errC)
	}()
	stop := ctx.Done()
	for {
		select {
		case <-stop:
			stop = nil
			ctx, canc := context.WithTimeout(context.Background(), 30*time.Second)
			if err := s.server.Shutdown(ctx); err != nil {
				log.Warnf("error shutting down: %s", err)
			}
			canc()
		case err := <-errC:
			if err != nil {
				log.Warnf("Error serving HTTP: %s", err.Error())
			}
			close(s.done)
			return
		}
	}
}

func (s *HttpServer) Done() <-chan struct{} {
	return s.done
}
