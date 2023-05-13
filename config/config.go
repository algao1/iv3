package config

type Config struct {
	Dexcom  DexcomConfig    `yaml:"dexcom"`
	Insulin []InsulinConfig `yaml:"insulin"`
	API     APIConfig       `yaml:"api"`
	Spaces  SpacesConfig    `yaml:"spaces"`
	Alert   AlertConfig     `yaml:"alert"`
}

type DexcomConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}

type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type SpacesConfig struct {
	Key    string `yaml:"key"`
	Secret string `yaml:"secret"`
}

type InsulinConfig struct {
	Name     string  `yaml:"name"`
	Duration int     `yaml:"duration"`
	Peak     float64 `yaml:"peak"`
	Type     string  `yaml:"type"`
}

type AlertConfig struct {
	Endpoint string `yaml:"endpoint"`
}
