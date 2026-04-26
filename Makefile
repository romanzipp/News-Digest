.PHONY: dev build run css css-watch setup

setup:
	npm install
	cp node_modules/htmx.org/dist/htmx.min.js static/js/htmx.min.js
	mkdir -p static/fonts
	cp node_modules/@fontsource/playfair-display/files/playfair-display-latin-700-normal.woff2 static/fonts/
	cp node_modules/@fontsource/playfair-display/files/playfair-display-latin-900-normal.woff2 static/fonts/
	cp node_modules/@fontsource/source-serif-4/files/source-serif-4-latin-400-normal.woff2 static/fonts/
	cp node_modules/@fontsource/source-serif-4/files/source-serif-4-latin-400-italic.woff2 static/fonts/
	cp node_modules/@fontsource/source-serif-4/files/source-serif-4-latin-600-normal.woff2 static/fonts/

dev:
	@make -j2 css-watch run

run:
	go run ./cmd/news

build:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --minify
	CGO_ENABLED=1 go build -o bin/news ./cmd/news

css:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --minify

css-watch:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --watch
