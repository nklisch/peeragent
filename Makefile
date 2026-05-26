.PHONY: build test clean

build:
	./scripts/build.sh

test:
	go test ./...

clean:
	rm -rf dist
