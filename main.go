package main

import (
	"flag"
	"os"

	"github.com/algao1/iv3/fetcher"
	"github.com/algao1/iv3/store"
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

func main() {
	// Probably will use a mix of flags and config files.
	// Though this might get overwhelming/confusing, for now it should be ok.
	logger, _ := zap.NewDevelopment()
	verifyFlags(logger)

	file, err := os.ReadFile(configFile)
	if err != nil {
		logger.Fatal("unable to read config file", zap.Error(err))
	}

	cfg := Config{}
	if err = yaml.Unmarshal(file, &cfg); err != nil {
		logger.Fatal("unable to unmarshal config file", zap.Error(err))
	}

	influxClient := store.NewInfluxDB(
		influxdbToken,
		influxdbUrl,
		logger.Named("influxdb"),
	)

	fetcher.NewDexcom(
		cfg.Dexcom.Account,
		cfg.Dexcom.Password,
		[]fetcher.GlucosePointsWriter{influxClient},
		logger.Named("dexcom"),
	)

	// Block.
	logger.Info("everything started successfully!")
	select {}
}
