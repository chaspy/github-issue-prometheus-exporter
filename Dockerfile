FROM golang:1.15.7 as builder

WORKDIR /go/src

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go  ./

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN go build \
    -o /go/bin/github-issue-prometheus-exporter \
    -ldflags '-s -w'

FROM alpine:3.13.0 as runner

COPY --from=builder /go/bin/github-issue-prometheus-exporter /app/github-issue-prometheus-exporter

ENTRYPOINT ["/app/github-issue-prometheus-exporter"]
