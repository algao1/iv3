package main

type Config struct {
	Dexcom DexcomConfig `yaml:"dexcom"`
	API    APIConfig    `yaml:"api"`
}

type DexcomConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}

type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
