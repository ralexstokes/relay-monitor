package main

import (
	"flag"
	"os"

	monitor "github.com/ralexstokes/relay-monitor/pkg"
	"gopkg.in/yaml.v3"
)

var (
	configFile = flag.String("config", "config.example.yaml", "path to config file")
)

func main() {
	flag.Parse()

	data, err := os.ReadFile(*configFile)
	if err != nil {
		// TODO log and exit
		panic(err)
	}

	config := &monitor.Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		// TODO log and exit
		panic(err)
	}

	monitor := monitor.New(config)
	monitor.Run()
}
