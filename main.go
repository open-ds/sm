package main

import (
	"flag"
	"github.com/open-ds/sm/lib"
)

var (
	config string
)

func main() {
	flag.StringVar(&config, "c", "./config/config.yaml", "config file")
	flag.Parse()

	server := lib.NewServer()
	server.InitConfig(config)
	server.Serve()
}
