package config

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	RateLimit            int
}

func NewConfig() *Config {
	flags := ParseFlags()

	return &Config{
		RunAddress:           flags.RunAddress,
		DatabaseURI:          flags.DatabaseURI,
		AccrualSystemAddress: flags.AccrualSystemAddress,
		RateLimit:            3,
	}
}
