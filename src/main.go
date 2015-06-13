package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/jannson/routercenter/src/server"
)

var cfgFile string

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	configPtr := flag.String("config", "", "config file")
	flag.Usage = usage
	flag.Parse()

	if *configPtr == "" {
		*configPtr = "./conf/config.ini"
	}

	cfgFile = *configPtr

	isExist, _ := exists(cfgFile)
	if !isExist {
		log.Fatal("config file not exist!")
		os.Exit(-1)
	}

	serverContext, err := rcenter.NewContext(cfgFile)
	if err != nil {
		log.Fatal("Error: ", err)
		os.Exit(-4)
	}
	defer serverContext.Release()

	rcenter.StartServer(serverContext)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:--config=/etc/config.ini \n")
	flag.PrintDefaults()
	os.Exit(-2)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
