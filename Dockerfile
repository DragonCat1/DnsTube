# DnsTube single image: embedded Vue build + Go binary (UDP DNS + HTTP)
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.22-alpine AS go
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./cmd/dnstube/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /dnstube ./cmd/dnstube

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go /dnstube /app/dnstube
ENV TZ=UTC
EXPOSE 8080/tcp
EXPOSE 5353/udp
# Default HTTP_ADDR is :8080; override HEALTHCHECK if you change HTTP_ADDR.
HEALTHCHECK --interval=30s --timeout=5s --start-period=25s --retries=3 \
  CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8080/healthz"]
ENTRYPOINT ["/app/dnstube"]
