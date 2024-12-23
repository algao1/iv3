package main

import (
	"flag"
	"os"

	"github.com/algao1/iv3/alert"
	"github.com/algao1/iv3/analysis"
	"github.com/algao1/iv3/config"
	"github.com/algao1/iv3/fetcher"
	"github.com/algao1/iv3/server"
	"github.com/algao1/iv3/store"
	"github.com/algao1/iv3/tools/auto_backup"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var (
	iv3Env        string
	configFile    string
	influxdbToken string
	influxdbUrl   string
)

func init() {
	flag.StringVar(&iv3Env, "iv3Env", "dev", "iv3 env var")
	flag.StringVar(&configFile, "config", "config.yaml", "config file")
	flag.StringVar(&influxdbToken, "influxdbToken", "", "InfluxDB token")
	flag.StringVar(&influxdbUrl, "influxdbUrl", "http://localhost:8086", "InfluxDB url")
	flag.Parse()
}

func verifyFlags(logger *zap.Logger) {
	if influxdbToken == "" {
		logger.Fatal("no InfluxDB token provided")
	}
}

func main() {
	logger, _ := zap.NewProduction()
	if iv3Env == "dev" {
		logger, _ = zap.NewDevelopment()
	}
	verifyFlags(logger)

	file, err := os.ReadFile(configFile)
	if err != nil {
		logger.Fatal("unable to read config file", zap.Error(err))
	}

	cfg := config.Config{}
	if err = yaml.Unmarshal(file, &cfg); err != nil {
		logger.Fatal("unable to unmarshal config file", zap.Error(err))
	}
	if err := cfg.Verify(); err != nil {
		logger.Fatal("incorrect config provided", zap.Error(err))
	}

	influxClient, err := store.NewInfluxDB(
		influxdbToken,
		influxdbUrl,
		logger.Named("influxdb"),
	)
	if err != nil {
		logger.Fatal("unable to create InfluxDB client", zap.Error(err))
	}

	backuper, err := auto_backup.NewS3Backuper(
		influxdbToken,
		influxdbUrl,
		cfg.Spaces,
		logger.Named("s3Backuper"),
	)
	if err != nil {
		logger.Fatal("unable to create S3 backuper", zap.Error(err))
	}

	if iv3Env != "dev" {
		logger.Info("starting S3 backuper")
		err := backuper.Start()
		if err != nil {
			logger.Fatal("unable to start S3 backuper", zap.Error(err))
		}
	}

	fetcher.NewDexcom(
		cfg.Dexcom.Account,
		cfg.Dexcom.Password,
		[]fetcher.GlucosePointsWriter{
			store.NewDDClient(&cfg.Iv3),
			influxClient,
		},
		logger.Named("dexcom"),
	)

	if cfg.Iv3.Endpoint != "" {
		alert.NewAlerter(
			influxClient,
			cfg.Iv3,
			cfg.Insulin,
			logger.Named("alerter"),
		)
	}

	analyzer := analysis.NewAnalyzer(
		influxClient,
		cfg.Iv3,
		logger.Named("analyzer"),
	)

	s := server.NewHttpServer(
		cfg.API.Username,
		cfg.API.Password,
		cfg,
		influxClient,
		analyzer,
		logger.Named("httpServer"),
	)

	logger.Info("everything started successfully!")
	s.Serve() // Blocking.
}
