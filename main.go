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

type PrometheusCachetParameters struct {
	loglevel            string
	httpPort            int
	sslCert             string
	sslKey              string
	cachetRootCA        string
	cachetSkipVerifySsl bool
	cachetURL           string
	cachetToken         string
	prometheusToken     string
	labelName           string
	squashIncident      bool
	cachetTimezone      string
}

// NewPrometheusCachetParameters is here to fetch all env variable or parameters
func NewPrometheusCachetParameters() *PrometheusCachetParameters {
	p := &PrometheusCachetParameters{}

	flag.StringVar(&p.prometheusToken, "prometheus_token", "", "token sent by Prometheus in the webhook configuration")
	flag.StringVar(&p.cachetURL, "cachethq_url", "http://127.0.0.1/", "where to find CachetHQ")
	flag.StringVar(&p.cachetToken, "cachethq_token", "", "token to send to CachetHQ")
	flag.StringVar(&p.cachetRootCA, "cachethq_root_ca", "", "Root SSL CA to use against CachetHQ")
	flag.BoolVar(&p.cachetSkipVerifySsl, "cachethq_skip_verify_ssl", false, "Dont check the SSL certificate of the https access to CachetHQ")
	flag.StringVar(&p.loglevel, "log_level", "info", "log level: [info|debug]")
	flag.StringVar(&p.sslCert, "ssl_cert_file", "", "to be used with ssl_key: enable https server")
	flag.StringVar(&p.sslKey, "ssl_key_file", "", "to be used with ssl_cert: enable https server")
	flag.StringVar(&p.labelName, "label_name", "alertname", "label to look for in Prometheus Alert info")
	flag.IntVar(&p.httpPort, "http_port", 8080, "port to listen on")
	flag.BoolVar(&p.squashIncident, "squash_incident", false, "do we want to merge down and up event into one incident")
	flag.StringVar(&p.cachetTimezone, "cachethq_timezone", "+0000", "Timezone configured in CachetHQ: UTC=>+0000, EST=>-0600, ...")
	flag.Parse()

	// grab env variable (docker compliant)
	if os.Getenv("CACHETHQ_TIMEZONE") != "" {
		p.cachetTimezone = os.Getenv("CACHETHQ_TIMEZONE")
	}
	if os.Getenv("PROMETHEUS_TOKEN") != "" {
		p.prometheusToken = os.Getenv("PROMETHEUS_TOKEN")
	}
	if os.Getenv("CACHETHQ_URL") != "" {
		p.cachetURL = os.Getenv("CACHETHQ_URL")
	}
	if os.Getenv("CACHETHQ_TOKEN") != "" {
		p.cachetToken = os.Getenv("CACHETHQ_TOKEN")
	}
	if os.Getenv("CACHETHQ_ROOT_CA") != "" {
		p.cachetRootCA = os.Getenv("CACHETHQ_ROOT_CA")
	}
	if os.Getenv("CACHETHQ_SKIP_VERIFY_SSL") == "true" {
		p.cachetSkipVerifySsl = true
	}
	if os.Getenv("LOG_LEVEL") != "" {
		p.loglevel = os.Getenv("LOG_LEVEL")
	}
	if os.Getenv("HTTP_PORT") != "" {
		if port, err := strconv.Atoi(os.Getenv("HTTP_PORT")); err == nil {
			p.httpPort = port
		}
	}
	if os.Getenv("SSL_CERT_FILE") != "" {
		p.sslCert = os.Getenv("SSL_CERT_FILE")
	}
	if os.Getenv("SSL_KEY_FILE") != "" {
		p.sslKey = os.Getenv("SSL_KEY_FILE")
	}

	if os.Getenv("LABEL_NAME") != "" {
		p.labelName = os.Getenv("LABEL_NAME")
	}

	if os.Getenv("SQUASH_INCIDENT") == "true" {
		p.squashIncident = true
	}
	return p
}

type PrometheusCachetConfig struct {
	PrometheusToken string
	Cachet          Cachet
	LabelName       string
	LogLevel        int
	SquashIncident  bool
	Timezone        string
}

func main() {
	parameters := NewPrometheusCachetParameters()

	caCertPool := x509.NewCertPool()
	if parameters.cachetRootCA != "" {
		caCert, err := ioutil.ReadFile(parameters.cachetRootCA)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool.AppendCertsFromPEM(caCert)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            caCertPool,
				InsecureSkipVerify: parameters.cachetSkipVerifySsl,
			},
		},
	}

	config := PrometheusCachetConfig{
		PrometheusToken: parameters.prometheusToken,
		Cachet:          NewCachetImpl(parameters.cachetURL, parameters.cachetToken, httpClient),
		LabelName:       parameters.labelName,
		LogLevel:        LOG_INFO,
		SquashIncident:  parameters.squashIncident,
		Timezone:        parameters.cachetTimezone,
	}

	config.LogLevel = LOG_INFO
	if parameters.loglevel == "debug" {
		config.LogLevel = LOG_DEBUG
	}

	router := PrepareGinRouter(&config)

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", parameters.httpPort),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if parameters.sslCert != "" && parameters.sslKey != "" {
		log.Fatal(server.ListenAndServeTLS(parameters.sslCert, parameters.sslKey))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
