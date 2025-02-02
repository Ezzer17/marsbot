FROM golang:1.24-rc-alpine AS builder
RUN apk add build-base
ADD src /app
WORKDIR /app
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build ./

FROM alpine:3.21.2
COPY --from=builder /app/marsbot /
RUN apk add bash
CMD ["/marsbot", "-config", "config.yaml"]
