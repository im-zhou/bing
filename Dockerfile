FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bing .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 bing
WORKDIR /app
COPY --from=build /bing /app/bing
USER bing
EXPOSE 12345
ENTRYPOINT ["/app/bing", "-config", "/app/config.yaml"]
