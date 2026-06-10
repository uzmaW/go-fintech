# ---- build stage ----
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata

ARG SERVICE
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /bin/service ./cmd/${SERVICE}

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/service /service
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/service"]
