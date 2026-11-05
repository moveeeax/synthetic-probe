FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /synthetic-probe .

# gcr.io/distroless/static:nonroot, pinned by digest (resolved 2026-11-05).
# Dependabot (docker ecosystem) proposes digest bumps.
FROM gcr.io/distroless/static:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
COPY --from=build /synthetic-probe /synthetic-probe
COPY examples/targets.yaml /etc/synthetic-probe/targets.yaml
EXPOSE 9115
ENTRYPOINT ["/synthetic-probe"]
CMD ["--config", "/etc/synthetic-probe/targets.yaml", "--listen", ":9115"]
