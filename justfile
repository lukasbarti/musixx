set shell := ["pwsh.exe", "-NoLogo", "-Command"]

default: build

build:
	go build ./...

run:
	go tool air --build.pre_cmd="just generate" --build.include_ext="css,js,templ,go" --build.exclude_regex="_templ.go" --build.args_bin="serve"

test:
	go tool gotestsum

fmt:
	gofmt -w cmd internal main.go

generate:
	go tool templ generate
