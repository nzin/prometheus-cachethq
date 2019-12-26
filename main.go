package prometheuscachethq

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
	Cachet          Cachet
	LabelName       string
	LogLevel        int
}

func main() {
	var loglevel string
	var httpPort int
	var sslCert string
	var sslKey string
	var cachetRootCA string
	var cachetSkipVerifySsl bool
	var cachetURL string
	var cachetToken string
	var prometheusToken string
	var labelName string
	flag.StringVar(&prometheusToken, "prometheus_token", "", "token sent by Prometheus in the webhook configuration")
	flag.StringVar(&cachetURL, "cachethq_url", "http://127.0.0.1/", "where to find CachetHQ")
	flag.StringVar(&cachetToken, "cachethq_token", "", "token to send to CachetHQ")
	flag.StringVar(&cachetRootCA, "cachethq_root_ca", "", "Root SSL CA to use against CachetHQ")
	flag.BoolVar(&cachetSkipVerifySsl, "cachethq_skip_verify_ssl", false, "Dont check the SSL certificate of the https access to CachetHQ")
	flag.StringVar(&loglevel, "log_level", "info", "log level: [info|debug]")
	flag.StringVar(&sslCert, "ssl_cert_file", "", "to be used with ssl_key: enable https server")
	flag.StringVar(&sslKey, "ssl_key_file", "", "to be used with ssl_cert: enable https server")
	flag.StringVar(&labelName, "label_name", "alertname", "label to look for in Prometheus Alert info")
	flag.IntVar(&httpPort, "http_port", 8080, "port to listen on")

	flag.Parse()

	// grab env variable (docker compliant)
	if os.Getenv("PROMETHEUS_TOKEN") != "" {
		prometheusToken = os.Getenv("PROMETHEUS_TOKEN")
	}
	if os.Getenv("CACHETHQ_URL") != "" {
		cachetURL = os.Getenv("CACHETHQ_URL")
	}
	if os.Getenv("CACHETHQ_TOKEN") != "" {
		cachetToken = os.Getenv("CACHETHQ_TOKEN")
	}
	if os.Getenv("CACHETHQ_ROOT_CA") != "" {
		cachetRootCA = os.Getenv("CACHETHQ_ROOT_CA")
	}
	if os.Getenv("CACHETHQ_SKIP_VERIFY_SSL") == "true" {
		cachetSkipVerifySsl = true
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
		labelName = os.Getenv("LABEL_NAME")
	}

	caCertPool := x509.NewCertPool()
	if cachetRootCA != "" {
		caCert, err := ioutil.ReadFile(cachetRootCA)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool.AppendCertsFromPEM(caCert)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            caCertPool,
				InsecureSkipVerify: cachetSkipVerifySsl,
			},
		},
	}

	config := PrometheusCachetConfig{
		PrometheusToken: prometheusToken,
		Cachet:          NewCachetImpl(cachetURL, cachetToken, httpClient),
		LabelName:       labelName,
		LogLevel:        LOG_INFO,
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
