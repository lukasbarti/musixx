set shell := ["pwsh.exe", "-NoLogo", "-Command"]

default: build

bundle-js:
	esbuild web/assets/src/app.js --minify --bundle --outfile=web/assets/app.dist.js --format=iife --target=es2019 --sourcemap

bundle-css:
	npx @tailwindcss/cli -i ./web/assets/src/app.css -o ./web/assets/app.css --minify

build: bundle-js bundle-css
	go build ./...

run:
	go tool air --build.pre_cmd="just generate" --build.include_ext="css,js,templ,go" --build.exclude_regex="(_templ.go|.dist.js)" --build.args_bin="serve"

test:
	go tool gotestsum

fmt:
	gofmt -w cmd internal main.go

generate: bundle-js bundle-css
	go tool templ generate
