# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine3.24@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build
WORKDIR /src
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/loopaware ./cmd/server

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
WORKDIR /app
RUN apk add --no-cache ca-certificates && \
    mkdir -p /app/data /app/configs
COPY --from=build /out/loopaware /app/loopaware
EXPOSE 8080
ENTRYPOINT ["/app/loopaware"]
