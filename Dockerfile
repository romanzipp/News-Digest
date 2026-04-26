FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev nodejs npm

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

RUN cp node_modules/htmx.org/dist/htmx.min.js static/js/htmx.min.js
RUN mkdir -p static/fonts && \
    cp node_modules/@fontsource/playfair-display/files/playfair-display-latin-700-normal.woff2 static/fonts/ && \
    cp node_modules/@fontsource/playfair-display/files/playfair-display-latin-900-normal.woff2 static/fonts/ && \
    cp node_modules/@fontsource/source-serif-4/files/source-serif-4-latin-400-normal.woff2 static/fonts/ && \
    cp node_modules/@fontsource/source-serif-4/files/source-serif-4-latin-400-italic.woff2 static/fonts/ && \
    cp node_modules/@fontsource/source-serif-4/files/source-serif-4-latin-600-normal.woff2 static/fonts/
RUN npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --minify
RUN CGO_ENABLED=1 go build -o /news ./cmd/news

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /news /app/news
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/static /app/static

EXPOSE 8080

CMD ["/app/news"]
