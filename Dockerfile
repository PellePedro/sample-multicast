FROM golang as builder

# Build Arguments
ARG build
ARG version

WORKDIR /src

COPY . /src

RUN   go mod download \
      && CGO_ENABLED=0 go build \
      -ldflags="-s -w -X main.Version=${version} -X main.Build=${build}" -o /halo cmd/app/main.go

FROM ubuntu:20.04
COPY --from=builder /halo /halo

EXPOSE 86

ENTRYPOINT ["./halo"]
