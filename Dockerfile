# ── Stage 1: Build frontend assets ──────────────────────────────────────────
FROM node:22-alpine AS frontend
WORKDIR /build

COPY package.json package-lock.json ./
RUN npm ci

COPY web/assets/src ./web/assets/src
RUN npx esbuild web/assets/src/app.js \
      --minify --bundle \
      --outfile=web/assets/app.dist.js \
      --format=iife --target=es2019
RUN npx @tailwindcss/cli \
      -i ./web/assets/src/app.css \
      -o ./web/assets/app.css \
      --minify

# ── Stage 2: Build Go binary ─────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /build/web/assets/app.dist.js ./web/assets/app.dist.js
COPY --from=frontend /build/web/assets/app.css     ./web/assets/app.css

RUN go tool templ generate
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o musixx .

# ── Stage 3: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/musixx           ./musixx
COPY --from=builder /build/web/assets/css   ./web/assets/css
COPY --from=builder /build/web/assets/app.css            ./web/assets/app.css
COPY --from=builder /build/web/assets/app.dist.js        ./web/assets/app.dist.js
COPY --from=builder /build/web/assets/soundtouch-processor.js ./web/assets/soundtouch-processor.js

RUN mkdir -p /app/music /app/data

VOLUME ["/app/music", "/app/data"]
EXPOSE 8080

ENV MUSIXX_STORAGE_DATABASE_PATH=/app/data/musixx.db \
    MUSIXX_LIBRARY_MUSIC_DIR=/app/music

ENTRYPOINT ["./musixx"]
CMD ["serve"]
