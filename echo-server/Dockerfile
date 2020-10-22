FROM golang:1.14

RUN mkdir -p /build
WORKDIR /build/
COPY . .

# CGO_ENABLED must be disabled to run go binary in Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -o bin/echo-server github.com/LaCumbancha/backup-server/echo-server

FROM busybox:latest
COPY --from=0 /build/bin/echo-server /echo-server
ENTRYPOINT ["/bin/sh"]
