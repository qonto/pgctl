FROM golang:1.24-alpine3.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

HEALTHCHECK NONE

RUN make build-alpine


FROM alpine:3.23

ARG USER=app
ARG HOME=/app

RUN addgroup -g 1001 -S app \
    && adduser --home /app -u 1001 -S app -G app \
    && mkdir -p /app \
    && chown app:app -R /app

WORKDIR $HOME
USER $USER

COPY --from=builder /build/bin/pgctl $HOME/pgctl

ENTRYPOINT [ "/app/pgctl" ]
