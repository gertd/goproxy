FROM golang:1.14-alpine AS builder
RUN apk add --no-cache bash build-base git tree
ADD . /src
RUN cd /src && make build

FROM alpine
WORKDIR /app
COPY --from=builder /src/bin/linux-amd64/goproxy /app/
ENTRYPOINT [ "./goproxy" ]