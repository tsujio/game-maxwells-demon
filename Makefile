title := $(shell grep '^module' go.mod | sed -e 's/.*\/game-\(.*\)$$/\1/')

.PHONY: all deploy

all:
	go generate resources/generate.go
	GOOS=js GOARCH=wasm go build -o $(title).wasm github.com/tsujio/game-$(title)
	gzip -c $(title).wasm > $(title).wasm.gz

deploy:
	gsutil -h "Content-Type:application/wasm" -h "Content-Encoding:gzip" cp $(title).wasm.gz gs://tsujio-game-serve/$(title)/
