package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	LOG_DEBUG = 0
	LOG_INFO  = 1
)

type PrometheusCachetConfig struct {
	PrometheusToken string
	CachetURL       string
	CachetToken     string
	LabelName       string
	LogLevel        int
	HttpClient      *http.Client
}

func main() {
	var config PrometheusCachetConfig
	var loglevel string
	var httpPort int
	var sslCert string
	var sslKey string
	var cachetRootCA string
	flag.StringVar(&config.PrometheusToken, "prometheus_token", "", "token sent by Prometheus in the webhook configuration")
	flag.StringVar(&config.CachetURL, "cachethq_url", "http://127.0.0.1/", "where to find CachetHQ")
	flag.StringVar(&config.CachetToken, "cachethq_token", "", "token to send to CachetHQ")
	flag.StringVar(&cachetRootCA, "cachethq_root_ca", "", "Root SSL CA to use against CachetHQ")
	flag.StringVar(&loglevel, "log_level", "info", "log level: [info|debug]")
	flag.StringVar(&sslCert, "ssl_cert_file", "", "to be used with ssl_key: enable https server")
	flag.StringVar(&sslKey, "ssl_key_file", "", "to be used with ssl_cert: enable https server")
	flag.StringVar(&config.LabelName, "label_name", "alertname", "label to look for in Prometheus Alert info")
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
	if os.Getenv("CACHETHQ_ROOT_CA") != "" {
		cachetRootCA = os.Getenv("CACHETHQ_ROOT_CA")
	}
	if os.Getenv("LOG_LEVEL") != "" {
		loglevel = os.Getenv("LOG_LEVEL")
	}
	if os.Getenv("HTTP_PORT") != "" {
		if port, err := strconv.Atoi(os.Getenv("HTTP_PORT")); err == nil {
			httpPort = port
		}
	}
	if os.Getenv("SSL_CERT_FILE") != "" {
		sslCert = os.Getenv("SSL_CERT_FILE")
	}
	if os.Getenv("SSL_KEY_FILE") != "" {
		sslKey = os.Getenv("SSL_KEY_FILE")
	}

	if os.Getenv("LABEL_NAME") != "" {
		config.LabelName = os.Getenv("LABEL_NAME")
	}

	caCertPool := x509.NewCertPool()
	if cachetRootCA != "" {
		caCert, err := ioutil.ReadFile(cachetRootCA)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool.AppendCertsFromPEM(caCert)
	}

	config.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	config.LogLevel = LOG_INFO
	if loglevel == "debug" {
		config.LogLevel = LOG_DEBUG
	}
	router := PrepareGinRouter(&config)

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", httpPort),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if sslCert != "" && sslKey != "" {
		log.Fatal(server.ListenAndServeTLS(sslCert, sslKey))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
