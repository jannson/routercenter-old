package rcenter

import (
	"fmt"
)

var contextGlobal *ServerContext

type ServerContext struct {
	Config AppConfig
	Logger *ServerLogger
}

func NewContext(configFile string) (*ServerContext, error) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		return nil, err
	}

	var log *ServerLogger
	logOutput := cfg.System.LogOutput
	if logOutput == "file" {
		log, err = NewFileLogger("rcenter", 0, cfg.System.LogName)
	} else if logOutput == "console" {
		log, err = NewLogger("rcenter", 0)
	} else {
		return nil, fmt.Errorf("init logger failed")
	}

	if err != nil {
		return nil, err
	}

	contextGlobal = &ServerContext{Config: cfg, Logger: log}
	return contextGlobal, nil
}

func GetContextGlobal() *ServerContext {
	return contextGlobal
}

func (s *ServerContext) Release() {
	s.Logger.Close()
}
