.PHONY: build test validate clean

build:
	./scripts/build.sh

test:
	go test ./...

validate:
	./scripts/validate.sh

clean:
	rm -rf dist
