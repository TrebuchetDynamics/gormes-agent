# Gormes OCI image: Go-binary runtime, no Python/Node/Playwright deps.
# Two-stage build keeps the final image to a single static gormes binary
# plus the entrypoint shell script and a config example.

FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gormes ./cmd/gormes

FROM busybox:1.36.1

RUN mkdir -p /opt/gormes/docker /opt/data /etc/ssl/certs \
    && chown -R 65534:65534 /opt/gormes /opt/data

COPY --from=build /out/gormes /usr/local/bin/gormes
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chown=65534:65534 docker/entrypoint.sh /opt/gormes/docker/entrypoint.sh

ENV GORMES_HOME=/opt/data
VOLUME ["/opt/data"]
USER 65534:65534

# divergence: hosted-honcho compose stack is docs-only, not a runtime dep.
# divergence: honcho docker-compose.yml.example -> docs operational example.
# divergence: honcho docker/prometheus.yml -> docs operational example.
# divergence: honcho docker/grafana-datasource.yml -> docs operational example.

ENTRYPOINT ["/opt/gormes/docker/entrypoint.sh"]
CMD ["doctor", "--offline"]
