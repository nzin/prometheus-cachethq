# Prometheus Configuration

You have to configure the Prometheus Alert Manager something like:

    route:
      ...
        routes:
        - match:
            service: myservice
          receiver: cachethq-receiver

      receivers:
      - name: cachethq-receiver
        webhook_configs:
        - url: http://prometheus_cachet_bridge:8080/alert
          http_config:
            bearer_token: _prometheus_bearer_token_

# Prometheus CachetHQ bridge

If you have a CachetHQ and a Prometheus running on your local machine

    go build .
    ./prometheus-cachethq -prometheus_token _prometheus_bearer_token_ -cachethq_token _token_

    # to test, you can send by hand an alert to the Prometheus Alert Manager
    curl -H "Content-Type: application/json" -d '[{"labels":{"alertname":"component21"}}]' localhost:9093/api/v1/alerts

# Running as https

You need to provide a ssl cert AND a ssl key file:
    
    # Key considerations for algorithm "RSA" ≥ 2048-bit
    openssl genrsa -out server.key 2048

    # Key considerations for algorithm "ECDSA" ≥ secp384r1
    # List ECDSA the supported curves (openssl ecparam -list_curves)
    openssl ecparam -genkey -name secp384r1 -out server.key

Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based on the private (.key)

    openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650

And you can start the bridge like:

    ./prometheus-cachethq -prometheus_token _prometheus_bearer_token_ -cachethq_token _token_ -ssl_cert_file ./server.crt --ssl_key_file ./server.key
    
# Running with Docker / Kubernetes


