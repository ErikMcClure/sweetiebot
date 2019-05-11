# Use builder image to build and do initial config
FROM golang:alpine as builder
ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
ADD . /go
RUN apk add --no-cache git
RUN go get github.com/blackhole12/discordgo
RUN cd src/github.com/blackhole12/discordgo/;git checkout develop
RUN go get github.com/go-sql-driver/mysql
RUN go get "4d63.com/tz"
RUN go get "golang.org/x/net/idna"
RUN go get "golang.org/x/crypto/nacl/secretbox"
RUN go build -a -installsuffix cgo -o sweetie.out ./sweetie
RUN go build -a -installsuffix cgo -o updater.out ./updater

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/sweetie.out /sweetie
COPY --from=builder /go/updater.out /updater
ADD data /
ADD sweetiebot.sql /
ADD sweetiebot_tz.sql /
ADD web.css /
ADD web.html /
ADD docker_run.sh /
RUN ["chmod", "+x", "/docker_run.sh"]
RUN ["chmod", "+x", "/sweetie"]
RUN ["chmod", "+x", "/updater"]
CMD ["./docker_run.sh"]