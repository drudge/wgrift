# syntax=docker/dockerfile:1
# GoReleaser runtime image — binary is pre-built and provided by GoReleaser
FROM alpine:3.21

RUN apk add --no-cache wireguard-tools iptables ca-certificates && \
    mkdir -p /etc/wgrift /var/lib/wgrift /etc/wireguard

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/wgrift /usr/local/bin/wgrift
COPY deploy/docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY deploy/config.yaml /etc/wgrift/config.yaml.default

RUN chmod +x /usr/local/bin/wgrift /usr/local/bin/docker-entrypoint.sh

VOLUME ["/etc/wgrift", "/var/lib/wgrift", "/etc/wireguard"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["serve", "--config", "/etc/wgrift/config.yaml"]
