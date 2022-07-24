package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/sirupsen/logrus"
)

type HttpCollector struct {
	msgTopic  string
	failTopic string
	access    []net.IPNet
	l         *net.TCPListener
}

func newHttpCollector(listen string, access []net.IPNet, msgTopic, failTopic string) (*HttpCollector, error) {
	addr, err := net.ResolveTCPAddr("tcp", listen)
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &HttpCollector{
		msgTopic:  msgTopic,
		failTopic: failTopic,
		access:    access,
		l:         l,
	}, nil
}

func (c *HttpCollector) run(ctx context.Context, msgC chan<- *kafka.Message) {
	var (
		errC = make(chan error)
		stop = ctx.Done()
	)
	srv := &http.Server{
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           c.handler(msgC),
	}
	go func() {
		defer func() {
			log.Info("exited")
			close(errC)
		}()
		log.Infof("Start listening HTTP on %s", c.l.Addr().String())
		if err := srv.Serve(c.l); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errC <- err
			}
		}
	}()
	for {
		select {
		case <-stop:
			stop = nil
			sCtx, canc := context.WithTimeout(context.Background(), 5*time.Second)
			if err := srv.Shutdown(sCtx); err != nil {
				log.Warnf("error shutting down: %s", errors.Unwrap(err).Error())
			}
			canc()
		case err := <-errC:
			if err != nil {
				log.Warnf("error serving: %s", errors.Unwrap(err).Error())
			}
			return
		}
	}
}

func getAccessList(list []string) ([]net.IPNet, error) {
	addrs := make([]net.IPNet, 0, len(list))
	for _, s := range list {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, *n)
	}
	if len(addrs) == 0 {
		return nil, nil
	}
	return addrs, nil
}
