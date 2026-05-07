FROM golang:1.25 AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY="https://goproxy.cn,direct"
WORKDIR /opt/blog/src/
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -a -installsuffix cgo -o blog .

FROM alpine:latest
RUN apk add --no-cache ca-certificates git
WORKDIR /opt/blog/src/
COPY --from=builder /opt/blog/src/blog .
COPY --from=builder /opt/blog/src/tpl/ tpl/
COPY --from=builder /opt/blog/src/static/ static/
COPY --from=builder /opt/blog/src/conf/ conf/
COPY --from=builder /opt/blog/src/robots.txt .
EXPOSE 9091
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:9091/ping || exit 1
CMD ["./blog", "-config", "conf/prod.yaml"]
