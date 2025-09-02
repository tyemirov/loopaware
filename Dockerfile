# syntax=docker/dockerfile:1
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/feedbacksvc ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/feedbacksvc /app/feedbacksvc
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/feedbacksvc"]
