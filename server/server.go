package server

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/dmkorzh/test-project/server/handlers"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type HttpServer struct {
	server   http.Server
	listener net.Listener
	ctrl     handlers.Controller
	log      *logrus.Logger
	done     chan struct{}
}

//go:embed sql
var content embed.FS

func NewServer(db *sql.DB, addr, msgTopic, failTopic string, accessList []net.IP) (*HttpServer, error) {
	err := db.Ping()
	if err != nil {
		return nil, err
	}

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
			AccessList: accessList,
			MsgTopic:   msgTopic,
			FailTopic:  failTopic,
			Database:   &handlers.Database{Driver: db, Table: "calls"},
		},
		done: make(chan struct{}),
	}
	serv.makeRouter(router)
	serv.ctrl.Templates, err = parseTemplates(router)
	return serv, nil
}

func (s *HttpServer) Serve(ctx context.Context, msgC chan *kafka.Message) {
	errC := make(chan error)
	s.ctrl.MsgC = msgC
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
				s.log.Warnf("error shutting down: %s", err)
			}
			canc()
		case err := <-errC:
			if err != nil {
				s.log.Warnf("Error serving HTTP: %s", err.Error())
			}
			close(s.done)
			return
		}
	}
}

func (s *HttpServer) Done() <-chan struct{} {
	return s.done
}
