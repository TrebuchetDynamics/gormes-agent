# Gormes OCI image: Go-binary runtime, no Python/Node/Playwright deps.
# Two-stage build keeps the final image to a single static gormes binary
# plus the entrypoint shell script and a config example.

FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gormes ./cmd/gormes

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/gormes /usr/local/bin/gormes
COPY docker/entrypoint.sh /opt/gormes/docker/entrypoint.sh

ENV GORMES_HOME=/opt/data
VOLUME ["/opt/data"]

# divergence: hosted-honcho compose stack is docs-only, not a runtime dep.
# divergence: honcho docker-compose.yml.example -> docs operational example.
# divergence: honcho docker/prometheus.yml -> docs operational example.
# divergence: honcho docker/grafana-datasource.yml -> docs operational example.

ENTRYPOINT ["/opt/gormes/docker/entrypoint.sh"]
CMD ["doctor", "--offline"]
