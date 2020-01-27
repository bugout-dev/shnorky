test_target ?= ./...

build: shnorky

format:
	gofmt -s -w .

test:
	go test $(cli_args) $(test_target)

clean:
	touch shnorky
	rm shnorky

shnorky:
	CGO_ENABLED=1 go build $(cli_args) .

.PHONY: build test format clean
