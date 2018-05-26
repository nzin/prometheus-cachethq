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

You can either compile the Docker image (cf Dockerfile), or docker image on docker hub (nzin/prometheus-cachethq)

There is a minikube directory for an example. If you use this example:

- change the API_KEY to a random value in minikube/cachethq.yaml
- change the Prometheus token with yours in minikube/cachethq.yaml
- run the application (kubectl apply -f ./minikube)
- connect to cachethq to configure it
- go to http://<your minikube ip>:30080/dashboard/components/add to add a component (for example one called 'component21')
- go to http://<your minikube ip>:30080/dashboard/user to create a prometheus-cachethq user, and grab its token
- change the cachethq token in minikube/cachethq.yaml and restart the application (kubectl apply -f minikube/cachethq.yaml)

You can now simulate Prometheus Alertmanager by issuing something like:

    curl -X POST http://<minikube>:30081/alert -H 'Authorization: Bearer <prometheus token>' -d '{"receiver":"cachethq-receiver","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"component21"},"annotations":{},"startsAt":"2018-05-22T20:00:32.729840058-04:00","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":{"alertname":"component21"},"commonLabels":{"alertname":"component21"},"commonAnnotations":{},"externalURL":"http://localhost.localdomain:9093","version":"4","groupKey":"{}:{alertname=\"component21\"}"}'

