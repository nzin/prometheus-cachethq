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
            bearer_token: aaaaaaaa

# Prometheus CachetHQ bridge

    ./prometheus-cachethq -prometheus_token aaaaaaaa -cachethq_token _token_

    curl -H "Content-Type: application/json" -d '[{"labels":{"alertname":"component21"}}]' localhost:9093/api/v1/alerts

