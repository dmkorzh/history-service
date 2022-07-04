package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/dmkorzh/test-project/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	KEY_KAFKA               = "kafka"
	KEY_TOPIC               = "kafka.topics.calls"
	KEY_TOPIC_FAILED        = "kafka.topics.failed"
	KEY_CLICKHOUSE_HOSTS    = "clickhouse.hosts"
	KEY_CLICKHOUSE_USER     = "clickhouse.user"
	KEY_CLICKHOUSE_PASSWORD = "clickhouse.password"
	KEY_LISTEN              = "http.listen"
	KEY_ACCESS_LIST         = "http.accessList"
	KEY_LOG                 = "log"
)

func main() {
	viper.SetConfigType("yaml")
	cmd := &cobra.Command{
		Use:   "rest-service",
		Short: "REST service",
		RunE:  run,
	}
	cmd.Flags().StringP("config", "c", "/etc/rest-service/config.yaml", "Path for configuration file")
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config from \"%s\": %s", cfgPath, err.Error())
	}

	if err := setupLogging(); err != nil {
		return err
	}
	log.Infof("rest-service started with PID: %v, parameters: %s", os.Getppid(), os.Args[1:])
	defer log.Info("exited.")

	db, err := clickhouseConnect()
	if err != nil {
		log.Fatalf("Error connecting to clickhouse %s", err.Error())
		log.Exit(1)
	}
	log.Infof("Connected to clickhouse!")
	defer db.Close()

	listenAddr := viper.GetString(KEY_LISTEN)
	topic := viper.GetString(KEY_TOPIC)
	if topic == "" {
		return fmt.Errorf("no topic defined")
	}
	failTopic := viper.GetString(KEY_TOPIC_FAILED)
	accessList := getAccessList(viper.GetStringSlice(KEY_ACCESS_LIST))

	srv, err := server.NewServer(db, listenAddr, topic, failTopic, accessList)
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
		srv.Serve(ctx, msgC)
	}()
	awaitStop(srv, writerErrC, canc)
	return nil
}

func clickhouseConnect() (*sql.DB, error) {
	options := clickhouse.Options{
		Addr: viper.GetStringSlice(KEY_CLICKHOUSE_HOSTS),
		Auth: clickhouse.Auth{
			Database: "history",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 15,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}
	if viper.IsSet(KEY_CLICKHOUSE_USER) {
		options.Auth.Username = viper.GetString(KEY_CLICKHOUSE_USER)
		options.Auth.Password = viper.GetString(KEY_CLICKHOUSE_PASSWORD)
	}
	db := clickhouse.OpenDB(&options)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxIdleConns(64)
	db.SetMaxOpenConns(64)
	return db, nil
}

func getAccessList(list []string) []net.IP {
	addrs := make([]net.IP, 0, len(list))
	for _, s := range list {
		ip := net.ParseIP(s)
		addrs = append(addrs, ip)
	}
	if len(addrs) == 0 {
		return nil
	}
	return addrs
}

func kafkaConfig() (kafka.ConfigMap, error) {
	cfg := make(kafka.ConfigMap)
	kafkaKey := viper.Sub(KEY_KAFKA)
	brokers := kafkaKey.GetStringSlice("brokers")
	if brokers == nil {
		return nil, fmt.Errorf("No kafka broker specified!")
	}
	_ = cfg.SetKey("bootstrap.servers", strings.Join(brokers, ","))
	_ = cfg.SetKey("session.timeout.ms", 6000)
	_ = cfg.SetKey("go.logs.channel.enable", true)
	_ = cfg.SetKey("compression.type", "lz4")
	return cfg, nil
}

// Graceful shutdown
func awaitStop(srv *server.HttpServer, writerErrC <-chan error, canc context.CancelFunc) {
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
	case <-srv.Done():
		canc()
		return
	}
}
