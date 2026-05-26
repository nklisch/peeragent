.PHONY: build test validate release publish-release clean

build:
	./scripts/build.sh

test:
	go test ./...

validate:
	./scripts/validate.sh

release:
	./scripts/release.sh $(VERSION)

publish-release:
	./scripts/release.sh --publish $(VERSION)

clean:
	rm -rf dist
