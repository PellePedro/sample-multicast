FROM golang:1.17-alpine3.13 as builder

ADD . /go/src/halo

WORKDIR /go/src/halo

RUN apk add --no-cache git make build-base

RUN go mod download && \
    go build -gcflags="all=-N -l" \
	-ldflags="-X main.Version=1 -X main.Build=123 -X config.Build=123" -o /halo cmd/tsf/halo/main.go


FROM alpine:latest

COPY --from=builder /halo /halo

RUN apk add --update --no-cache \
    netcat-openbsd \
    bind-tools \
    curl \
    bash \
    darkhttpd \
    tcpdump \
    iperf3 \
    openssh \
    socat


CMD "/halo"
