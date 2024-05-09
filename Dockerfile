FROM golang:1.22.1 as builder
WORKDIR /src
COPY . .
RUN go mod tidy && \
    go build -o ./bin/insight-hub-rss -trimpath -ldflags '-s -w' github.com/ismdeep/insight-hub-rss

FROM debian:12
MAINTAINER "L. Jiang <l.jiang.1024@gmail.com>"
ENV TZ=Asia/Shanghai
RUN set -e; \
    apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y apt-transport-https ca-certificates tzdata
COPY --from=builder /src/bin/insight-hub-rss /usr/bin/
EXPOSE 8080
WORKDIR /data
ENTRYPOINT ["insight-hub-rss"]