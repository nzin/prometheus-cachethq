FROM golang:1.13-stretch as builder
LABEL maintainer "Nicolas Zin <nicolas.zin@gmail.com>"

RUN mkdir /app
WORKDIR /app

COPY . .
RUN set -x && \ 
    go mod download && \
    go test ./... && \
    CGO_ENABLED=0 GOOS=linux go build -a -o prometheus-cachethq


FROM alpine:3.7
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/prometheus-cachethq .

RUN chmod 755 prometheus-cachethq

#ENV PROMETHEUS_TOKEN
#ENV CACHETHQ_URL
#ENV CACHETHQ_TOKEN
#ENV SSL_CERT_FILE
#ENV SSL_KEY_FILE

EXPOSE 8080
ENTRYPOINT ["./prometheus-cachethq"]
