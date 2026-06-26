set shell := ["pwsh.exe", "-NoLogo", "-Command"]

default: build

bundle-js:
	esbuild web/assets/src/app.ts --minify --bundle --outfile=web/assets/app.dist.js --format=iife --target=es2019 --sourcemap

bundle-css:
	npx @tailwindcss/cli -i ./web/assets/src/app.css -o ./web/assets/app.css --minify

build: bundle-js bundle-css
	go build ./...

build-platform os arch: bundle-js bundle-css
	New-Item -ItemType Directory -Force dist | Out-Null
	$env:CGO_ENABLED = "0"; $env:GOOS = "{{os}}"; $env:GOARCH = "{{arch}}"; 
	go build -ldflags="-s -w" -trimpath -o "dist/musixx-{{os}}-{{arch}}{{ if os == "windows" { ".exe" } else { "" } }}" .

dist: (build-platform "linux" "amd64") (build-platform "linux" "arm64") (build-platform "darwin" "amd64") (build-platform "darwin" "arm64") (build-platform "windows" "amd64") (build-platform "windows" "arm64")

run:
	go tool air --build.pre_cmd="just generate" --build.include_ext="css,js,templ,go" --build.exclude_regex="(_templ.go|.dist.js)" --build.args_bin="serve"

test:
	go tool gotestsum

fmt:
	gofmt -w cmd internal main.go

generate: bundle-js bundle-css
	go tool templ generate
