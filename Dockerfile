FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /synthetic-probe .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /synthetic-probe /synthetic-probe
COPY examples/targets.yaml /etc/synthetic-probe/targets.yaml
EXPOSE 9115
ENTRYPOINT ["/synthetic-probe"]
CMD ["--config", "/etc/synthetic-probe/targets.yaml", "--listen", ":9115"]
