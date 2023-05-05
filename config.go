package main

type Config struct {
	Dexcom DexcomConfig `yaml:"dexcom"`
}

type DexcomConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}
