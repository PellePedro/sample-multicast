FROM golang as builder

# Build Arguments
ARG build
ARG version
ARG program=cmd/app/*.go

WORKDIR /src

COPY . /src

RUN   go mod download \
      && CGO_ENABLED=0 go build \
      -ldflags="-s -w -X main.Version=${version} -X main.Build=${build}" -o /halo ${program}

FROM scratch
COPY --from=builder /halo /halo

EXPOSE 89

ENTRYPOINT ["./halo"]
