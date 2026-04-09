# GoReleaser runtime image — binary is pre-built and provided by GoReleaser
FROM alpine:3.21

RUN apk add --no-cache wireguard-tools ca-certificates && \
    mkdir -p /etc/wgrift /var/lib/wgrift /etc/wireguard

COPY wgrift /usr/local/bin/wgrift
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY config.yaml /etc/wgrift/config.yaml.default

VOLUME ["/etc/wgrift", "/var/lib/wgrift", "/etc/wireguard"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["serve", "--config", "/etc/wgrift/config.yaml"]
