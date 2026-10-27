# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine3.24@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS build
WORKDIR /src
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/loopaware ./cmd/server

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
WORKDIR /app
RUN apk add --no-cache ca-certificates && \
    mkdir -p /app/data /app/configs
COPY --from=build /out/loopaware /app/loopaware
EXPOSE 8080
ENTRYPOINT ["/app/loopaware"]
