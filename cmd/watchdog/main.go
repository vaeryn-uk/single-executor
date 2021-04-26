package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"single-executor/internal/watchdog"
)

func main() {
	err, config := loadConfig()

	if err != nil {
		log.Printf("%s", err)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Loaded config: %+v", config)

	w := watchdog.NewWatchdog(config)

	err, errs := w.Start()

	if err != nil {
		log.Fatalf("Could not start watchdog: %s\n", err.Error())
	}

	for {
		// Simply loops on error channel and report them.
		log.Printf("Watchdog error: %s\n", (<- errs).Error())
	}
}

func loadConfig() (error, watchdog.Configuration) {
	var configFile string
	var configuration watchdog.Configuration

	flag.StringVar(&configFile, "c", "", "The watchdog config YAML file")
	flag.Parse()

	rawConfig, err := os.ReadFile(configFile)

	if len(configFile) == 0 {
		return fmt.Errorf("Must specify a configuration file"), configuration
	}

	if err != nil {
		return err, configuration
	}

	config, err := watchdog.ParseConfiguration(rawConfig)

	if err != nil {
		log.Fatalf("Invalid configuration: %s", err.Error())
	}

	return nil, config
}
