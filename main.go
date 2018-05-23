package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type PrometheusCachetConfig struct {
	PrometheusToken string
	CachetURL       string
	CachetToken     string
}

func main() {
	var config PrometheusCachetConfig
	var httpPort int
	flag.StringVar(&config.PrometheusToken, "prometheus_token", "", "token sent by Prometheus in the webhook configuration")
	flag.StringVar(&config.CachetURL, "cachethq_url", "http://127.0.0.1/", "where to find CachetHQ")
	flag.StringVar(&config.CachetToken, "cachethq_token", "", "token to send to CachetHQ")
	flag.IntVar(&httpPort, "http_port", 8080, "port to listen on")

	flag.Parse()

	// grab env variable (docker compliant)
	if os.Getenv("PROMETHEUS_TOKEN") != "" {
		config.PrometheusToken = os.Getenv("PROMETHEUS_TOKEN")
	}
	if os.Getenv("CACHETHQ_URL") != "" {
		config.CachetURL = os.Getenv("CACHETHQ_URL")
	}
	if os.Getenv("CACHETHQ_TOKEN") != "" {
		config.CachetToken = os.Getenv("CACHETHQ_TOKEN")
	}
	if os.Getenv("HTTP_PORT") != "" {
		if port, err := strconv.Atoi(os.Getenv("HTTP_PORT")); err == nil {
			httpPort = port
		}
	}

	router := PrepareGinRouter(&config)
	router.Run(fmt.Sprintf(":%d", httpPort))
}
