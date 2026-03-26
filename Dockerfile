# ---- Build stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/yeasy/mdpress/cmd.Version=${VERSION} -X github.com/yeasy/mdpress/cmd.BuildTime=${BUILD_TIME}" \
    -o /usr/local/bin/mdpress .

# ---- Minimal image (HTML, site, ePub — no PDF) ----
FROM alpine:3.21 AS minimal

RUN apk add --no-cache ca-certificates

# Run as non-root user for security best practices.
RUN addgroup -S mdpress && adduser -S mdpress -G mdpress

COPY --from=builder /usr/local/bin/mdpress /usr/local/bin/mdpress

# Default working directory for mounted book sources.
WORKDIR /book
RUN chown mdpress:mdpress /book

USER mdpress

ENTRYPOINT ["mdpress"]
CMD ["--help"]

# ---- Full image (all formats including PDF via Chromium) ----
FROM alpine:3.21 AS full

RUN apk add --no-cache \
    ca-certificates \
    chromium \
    font-noto \
    font-noto-cjk \
    font-noto-emoji

# Tell mdpress where to find Chromium for PDF rendering.
ENV MDPRESS_CHROME_PATH=/usr/bin/chromium-browser

# Run as non-root user for security best practices.
RUN addgroup -S mdpress && adduser -S mdpress -G mdpress

COPY --from=builder /usr/local/bin/mdpress /usr/local/bin/mdpress

WORKDIR /book
RUN chown mdpress:mdpress /book

USER mdpress

ENTRYPOINT ["mdpress"]
CMD ["--help"]
