package config

import (
	"fmt"
)

type Config struct {
	Dexcom  DexcomConfig    `yaml:"dexcom"`
	Insulin []InsulinConfig `yaml:"insulin"`
	API     APIConfig       `yaml:"api"`
	S3      S3Config        `yaml:"s3"`
	Iv3     Iv3Config       `yaml:"iv3"`
}

type DexcomConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}

type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type S3Config struct {
	Endpoint string `yaml:"endpoint"`
	Bucket   string `yaml:"bucket"`
	Key      string `yaml:"key"`
	Secret   string `yaml:"secret"`
}

type InsulinConfig struct {
	Name       string  `yaml:"name"`
	Duration   int     `yaml:"duration"`
	Peak       float64 `yaml:"peak"`
	PeriodType string  `yaml:"period_type"`
}

type Iv3Config struct {
	Unit                 string `yaml:"unit"`
	Endpoint             string `yaml:"endpoint"`
	MissingLongThreshold int    `yaml:"missing_long_threshold"`
	HighThreshold        int    `yaml:"high_threshold"`
	LowThreshold         int    `yaml:"low_threshold"`
}

func (cfg *Config) Verify() error {
	if cfg.API.Username == "" {
		return fmt.Errorf("no API username provided")
	}
	if cfg.API.Password == "" {
		return fmt.Errorf("no API password provided")
	}
	if cfg.Iv3.LowThreshold == 0 {
		cfg.Iv3.LowThreshold = 100
	}
	if cfg.Iv3.Unit != "mmol/L" && cfg.Iv3.Unit != "mg/dL" {
		return fmt.Errorf("incorrect unit provided: %s", cfg.Iv3.Unit)
	}

	return nil
}
