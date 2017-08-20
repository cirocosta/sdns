FROM golang:alpine as builder

ADD ./main.go /go/src/github.com/cirocosta/sdns/main.go
ADD ./lib /go/src/github.com/cirocosta/sdns/lib
ADD ./util /go/src/github.com/cirocosta/sdns/util
ADD ./vendor /go/src/github.com/cirocosta/sdns/vendor
ADD ./VERSION /go/src/github.com/cirocosta/sdns/VERSION

WORKDIR /go/src/github.com/cirocosta/sdns

RUN set -ex && \
  CGO_ENABLED=0 go build \
        -tags netgo -v -a \
        -ldflags "-X main.version=$(cat ./VERSION) -extldflags \"-static\"" && \
  mv ./sdns /usr/bin/sdns

FROM busybox
COPY --from=builder /usr/bin/sdns /usr/local/bin/sdns

ENTRYPOINT [ "sdns" ]

