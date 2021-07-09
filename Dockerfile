ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

COPY ./hubble-kafka-exporter /hubble-kafka-exporter

ENTRYPOINT [ "/hubble-kafka-exporter" ]
