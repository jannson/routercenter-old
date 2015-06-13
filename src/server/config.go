package rcenter

import (
	"code.google.com/p/gcfg"
)

type AppConfig struct {
	System struct {
		Host      string
		Port      int
		LogOutput string
		LogLevel  int
		LogName   string
	}
}

func LoadConfig(cfgFile string) (AppConfig, error) {
	var err error
	var cfg AppConfig

	err = gcfg.ReadFileInto(&cfg, cfgFile)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
