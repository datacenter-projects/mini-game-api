FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/apiserver . \
 && CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/migrate ./scripts/migrate

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -u 10001 app
ENV TZ=Asia/Bangkok
WORKDIR /app
COPY --from=builder /out/ /app/
USER app
EXPOSE 8181 9090
HEALTHCHECK --interval=15s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:8181/health/live || exit 1
ENTRYPOINT ["/app/apiserver"]
