package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"single-executor/internal/util"
	"single-executor/internal/watchdog"
	"strconv"
)

func main() {
	err, config, cluster := loadConfig()

	if err != nil {
		log.Printf("%s", err)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Loaded config: %+v\n", config)

	nodeIdEnv := util.MustGetEnv("NODE_ID")

	nodeId, err := strconv.Atoi(nodeIdEnv)

	if err != nil {
		log.Fatalf("Must specify numeric watchdog NODE_ID")
	}

	w := watchdog.NewWatchdog(watchdog.Id(nodeId), config, cluster)

	log.Printf("Starting debug HTTP server...\n")

	// Start an HTTP interface for debugging.
	go func() {
		if err := watchdog.HttpMonitor(w); err != nil {
			log.Fatalln(err)
		}
	}()

	log.Printf("Starting watchdog...\n")

	err = w.Start()

	if err != nil {
		log.Fatalf("Could not start watchdog: %s\n", err.Error())
	}

	log.Printf("Watchdog running...\n")

	for {
		// Simply loops on output channels and report details.
		select {
		case err := <- w.Errors:
			log.Printf("Watchdog ERR: %s\n", err.Error())
		case info := <- w.Info:
			log.Printf("Watchdog INFO %s\n", info)
		}
	}
}

func loadConfig() (error, watchdog.Configuration, watchdog.Cluster) {
	var configFile string
	var clusterFile string
	var config watchdog.Configuration
	var cluster watchdog.Cluster

	flag.StringVar(&configFile, "f", "", "The watchdog config YAML file")
	flag.StringVar(&clusterFile, "c", "", "The watchdog cluster YAML file")
	flag.Parse()

	if raw, err := readConfigFile(configFile, "Must specify config file"); err != nil {
		return err, config, cluster
	} else {
		if config, err = watchdog.ParseConfiguration(raw); err != nil {
			log.Fatalf("Invalid configuration: %s", err.Error())
		}
	}

	if raw, err := readConfigFile(clusterFile, "Must specify cluster file"); err != nil {
		return err, config, cluster
	} else {
		if cluster, err = watchdog.ParseCluster(raw); err != nil {
			log.Fatalf("Invalid configuration: %s", err.Error())
		}
	}

	return nil, config, cluster
}

func readConfigFile(file string, errmsg string) ([]byte, error) {
	raw, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("%s.\n", errmsg)
	}

	return raw, nil
}
