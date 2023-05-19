# syntax=docker/dockerfile:1

FROM golang as builder
ARG VERSION
WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

ADD . .
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -trimpath -ldflags "-s -X cmd.Version=$VERSION -X main.Version=$VERSION -linkmode external -extldflags '-static'" -v -o relay-monitor ./cmd/relay-monitor
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -trimpath -ldflags "-s -X cmd.Version=$VERSION -X main.Version=$VERSION -linkmode external -extldflags '-static'" -v -o website ./cmd/website

FROM alpine
RUN apk add --no-cache libstdc++ libc6-compat
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/relay-monitor /app/relay-monitor
COPY --from=builder /build/website /app/website
