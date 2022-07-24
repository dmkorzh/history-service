package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	. "github.com/dmkorzh/history-service/cmd"
	"github.com/dmkorzh/history-service/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigType("yaml")
	cmd := &cobra.Command{
		Use:   "rest-server",
		Short: "REST API server",
		RunE:  runServer,
	}
	cmd.Flags().StringP("config", "c", "config.yaml", "Path for configuration file")
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config from \"%s\": %s", cfgPath, err.Error())
	}

	if err := SetupLogger("rest-server-"); err != nil {
		return err
	}
	log.Infof("rest-server started with PID: %v, parameters: %s", os.Getppid(), os.Args[1:])
	defer log.Info("exited.")

	conn, err := clickhouseConnect()
	if err != nil {
		log.Fatal(err)
		log.Exit(1)
	}
	defer conn.Close()

	listenAddr := viper.GetString(KEY_REST_SERVER_ADDR)
	srv, err := server.NewServer(conn, listenAddr)
	if err != nil {
		log.Fatalf("Error listening on server %s", err.Error())
		log.Exit(1)
	}

	ctx, canc := context.WithCancel(context.Background())
	defer canc()
	go srv.Serve(ctx)
	go func() {
		<-srv.Done()
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigC
	signal.Reset()
	return nil
}

func clickhouseConnect() (clickhouse.Conn, error) {
	const timeout = 30 * time.Second
	options := &clickhouse.Options{
		Addr: viper.GetStringSlice(KEY_CLICKHOUSE_HOSTS),
		Auth: clickhouse.Auth{
			Database: viper.GetString(KEY_CLICKHOUSE_DB),
			Username: viper.GetString(KEY_CLICKHOUSE_USER),
			Password: viper.GetString(KEY_CLICKHOUSE_PASSWORD),
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    64,
		MaxIdleConns:    64,
		ConnMaxLifetime: 30 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time": 15,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}
	conn, _ := clickhouse.Open(options)
	if err := conn.Ping(context.Background()); err == nil {
		log.Infof("Start clickhouse connection on %v", options.Addr)
		return conn, nil
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn, _ := clickhouse.Open(options)
			if err := conn.Ping(context.Background()); err != nil {
				log.Warnf("DB ping error: %s, another try...", err.Error())
				continue
			}
			log.Infof("Start clickhouse connection on %v", options.Addr)
			return conn, nil
		case <-time.After(timeout):
			return nil, fmt.Errorf("DB connection timeout exceeded after %v!", timeout)
		}
	}
}
