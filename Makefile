.PHONY: build
build:
	earthly --use-inline-cache +watchclub \
		--GIT_COMMIT=$$(git rev-parse HEAD) \
		--VERSION=dev
	earthly --use-inline-cache +ui

gen-api:
	earthly --use-inline-cache  +proto

clean:
	rm -rf bin/
