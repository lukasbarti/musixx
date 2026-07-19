set windows-shell := ["pwsh.exe", "-NoLogo", "-Command"]

default: build

bundle-js:
	npx esbuild web/assets/src/app.ts --minify --bundle --outfile=web/assets/app.dist.js --format=iife --target=es2019 --sourcemap

bundle-css:
	cp web/assets/src/app.css web/assets/app.css

# Vendor Web Awesome's theme CSS and the icons we use. wa-icon fetches SVGs from the
# Font Awesome CDN by default; self-hosting keeps the app working offline.
wa_icons := "play pause stop backward-step forward-step bars xmark grip-lines"

bundle-wa:
	mkdir -p web/assets/css web/assets/wa/icons/solid
	npx esbuild node_modules/@awesome.me/webawesome/dist/styles/webawesome.css --bundle --minify --outfile=web/assets/css/webawesome.css
	for i in {{wa_icons}}; do cp node_modules/@fortawesome/fontawesome-free/svgs/solid/$i.svg web/assets/wa/icons/solid/; done

build: bundle-js bundle-css bundle-wa
	go build ./...

build-platform os arch: bundle-js bundle-css bundle-wa
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

generate: bundle-js bundle-css bundle-wa
	go tool templ generate
