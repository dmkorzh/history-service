package handlers

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/dmkorzh/test-project/parser"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func (ctrl *Controller) SaveCall(c *gin.Context) {
	addr := parseRemoteAddr(c.Request)
Check:
	for ctrl.AccessList != nil {
		for i := range ctrl.AccessList {
			if ctrl.AccessList[i].Equal(addr) {
				break Check
			}
		}
		err := fmt.Errorf("unexpected request from %s at %s", addr, c.Request.RequestURI)
		log.Warn(err)
		c.Writer.WriteHeader(http.StatusForbidden)
		_ = c.Error(err)
		return
	}
	body, err := ioutil.ReadAll(io.LimitReader(c.Request.Body, 1024*1024))
	if err != nil {
		log.Warn(err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		_ = c.Error(err)
		return
	}
	msg := &kafka.Message{
		Timestamp:     time.Now(),
		TimestampType: kafka.TimestampCreateTime,
		Headers: []kafka.Header{{
			Key:   "source",
			Value: []byte(c.Request.RemoteAddr),
		}},
		TopicPartition: kafka.TopicPartition{
			Partition: kafka.PartitionAny,
		},
	}
	if key, err := getKey(body); err == nil {
		call, err := parser.ParseCall(body)
		if err != nil {
			log.Warnf("failed to parse call: %s", err.Error())
			c.Writer.WriteHeader(http.StatusInternalServerError)
			_ = c.Error(err)
		}
		msg.TopicPartition.Topic = &ctrl.MsgTopic
		msg.Key = key
		msg.Value = call
		ctrl.MsgC <- msg
		c.Writer.WriteHeader(http.StatusOK)
		return
	} else if ctrl.FailTopic != "" {
		msg.TopicPartition.Topic = &ctrl.FailTopic
		msg.Value = body
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "error",
			Value: []byte(err.Error()),
		})
		ctrl.MsgC <- msg
		log.Warnf("failed to process message: %s", err.Error())
		c.Writer.WriteHeader(http.StatusInternalServerError)
		_ = c.Error(err)
	}
}

func parseRemoteAddr(r *http.Request) net.IP {
	if r.RemoteAddr == "" {
		return nil
	}
	if r.RemoteAddr[0] == '[' {
		if idx := strings.IndexByte(r.RemoteAddr[1:], ']'); idx != -1 {
			return net.ParseIP(r.RemoteAddr[1 : idx+1])
		} else {
			return nil
		}
	} else if idx := strings.IndexByte(r.RemoteAddr, ':'); idx != -1 {
		return net.ParseIP(r.RemoteAddr[:idx])
	} else {
		return net.ParseIP(r.RemoteAddr)
	}
}

func getKey(json []byte) ([]byte, error) {
	if !gjson.ValidBytes(json) {
		return nil, errors.New("invalid input JSON")
	}
	queueKey := gjson.GetBytes(json, "departmentID")
	if queueKey.Type != gjson.String || queueKey.Str == "" {
		return nil, errors.New("JSON has invalid department key")
	}
	return []byte(queueKey.Str), nil
}
