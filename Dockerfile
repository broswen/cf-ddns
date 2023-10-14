FROM golang:1.20 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /cf-ddns

FROM alpine:3.18.4
WORKDIR /
COPY --from=builder /cf-ddns /cf-ddns
USER nonroot:nonroot
ENTRYPOINT ["/cf-ddns"]