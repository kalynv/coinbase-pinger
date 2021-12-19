FROM golang:1.16 AS builder

WORKDIR /app
COPY go.mod go.sum /app
RUN go mod download
COPY *.go  /app
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o /webping

FROM scratch

COPY --from=builder /webping /webping
CMD ["/webping", "/prices/BTC-USD/buy"]
