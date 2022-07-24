package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	. "github.com/dmkorzh/history-service/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigType("yaml")
	cmd := &cobra.Command{
		Use:   "call",
		Short: "HTTP server for collecting calls",
		RunE:  runCollector,
	}
	cmd.Flags().StringP("config", "c", "config.yaml", "Path for configuration file")
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func runCollector(cmd *cobra.Command, args []string) error {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config from \"%s\": %s", cfgPath, err.Error())
	}

	if err := SetupLogger("calls-collector-"); err != nil {
		return err
	}
	log.Infof("calls-collector started with PID: %v, parameters: %s", os.Getppid(), os.Args[1:])
	defer log.Info("exited.")

	listenAddr := viper.GetString(KEY_COLLECTOR_ADDR)
	topic := viper.GetString(KEY_TOPIC)
	if topic == "" {
		return fmt.Errorf("no topic defined")
	}
	failTopic := viper.GetString(KEY_TOPIC_FAILED)
	accessList, err := getAccessList(viper.GetStringSlice(KEY_ACCESS_LIST))
	if err != nil {
		return err
	}
	srv, err := newHttpCollector(listenAddr, accessList, topic, failTopic)
	if err != nil {
		log.Fatalf("Error listening on server %s", err.Error())
		log.Exit(1)
	}

	msgC := make(chan *kafka.Message)
	kafkaConfig, err := kafkaConfig()
	if err != nil {
		return err
	}
	writerErrC, err := runKafkaWriter(msgC, &kafkaConfig)
	if err != nil {
		return err
	}

	ctx, canc := context.WithCancel(context.Background())
	defer canc()

	go func() {
		defer close(msgC)
		srv.run(ctx, msgC)
	}()

	awaitStop(writerErrC, canc)
	return nil
}

// Graceful shutdown
func awaitStop(writerErrC <-chan error, canc context.CancelFunc) {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
L:
	select {
	case <-sigC:
		canc()
		signal.Reset()
		sigC = nil
		goto L
	case err := <-writerErrC:
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}
