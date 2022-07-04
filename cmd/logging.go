package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

func setupLogging() error {
	var logKey *viper.Viper
	if viper.IsSet(KEY_LOG) {
		logKey = viper.Sub(KEY_LOG)
	} else {
		return errors.New("Logger not configured!")
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.999",
	})

	if logKey.IsSet("level") {
		level, err := logLevelFromText(logKey.GetString("level"))
		if err != nil {
			return errors.New("Unsupported log level!")
		}
		log.SetLevel(level)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if logKey.GetString("out") == "console" {
		log.SetOutput(os.Stdout)
		return nil
	}
	ts := time.Now()
	fileStamp := ts.Format("2006-01-02_15-04")
	if ts.Hour() == 0 && ts.Minute() == 0 {
		fileStamp = ts.Format("2006-01-02")
	}
	fileName := logKey.GetString("out") + "/" + fileStamp + ".log"
	rotateFile := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    logKey.GetInt("size"),
		MaxBackups: 10,
		MaxAge:     logKey.GetInt("age"),
		Compress:   true,
	}
	log.SetOutput(rotateFile)
	return nil
}

func logLevelFromText(level string) (log.Level, error) {
	m := map[string]log.Level{
		"fatal": log.FatalLevel,
		"warn":  log.WarnLevel,
		"info":  log.InfoLevel,
		"debug": log.DebugLevel,
	}
	l, ok := m[level]
	if !ok {
		return log.DebugLevel, fmt.Errorf("Invalid logging level: %s", level)
	}
	return l, nil
}

func logKafka(wg *sync.WaitGroup, producer *kafka.Producer) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for evt := range producer.Logs() {
			log.Infof("%s %s %s", evt.Name, evt.Tag, evt.Message)
		}
	}()
}
