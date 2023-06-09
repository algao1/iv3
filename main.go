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
	configFile    string
	influxdbToken string
	influxdbUrl   string
)

func init() {
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

func verifyConfig(cfg config.Config, logger *zap.Logger) {
	if cfg.API.Username == "" {
		logger.Fatal("no API username provided")
	}
	if cfg.API.Password == "" {
		logger.Fatal("no API password provided")
	}
	if cfg.Iv3.LowThreshold == 0 {
		cfg.Iv3.LowThreshold = 100
		logger.Info(
			"no low threshold provided, using default value",
			zap.Int("lowThreshold", cfg.Iv3.LowThreshold),
		)
	}
}

func main() {
	// TODO: should switch between prod and dev mode with flag.
	logger, _ := zap.NewProduction()
	verifyFlags(logger)

	file, err := os.ReadFile(configFile)
	if err != nil {
		logger.Fatal("unable to read config file", zap.Error(err))
	}

	cfg := config.Config{}
	if err = yaml.Unmarshal(file, &cfg); err != nil {
		logger.Fatal("unable to unmarshal config file", zap.Error(err))
	}
	verifyConfig(cfg, logger)

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

	iv3Env := os.Getenv("IV3_ENV")
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
		[]fetcher.GlucosePointsWriter{influxClient},
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
		influxClient,
		analyzer,
		logger.Named("httpServer"),
	)
	s.RegisterInsulin(cfg.Insulin)

	logger.Info("everything started successfully!")
	s.Serve() // Blocking.
}
