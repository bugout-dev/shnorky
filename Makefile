test_target ?= ./...

build: shn

format:
	gofmt -s -w .

test:
	go test $(cli_args) $(test_target)

clean:
	touch shn
	rm shn

shn:
	CGO_ENABLED=1 go build -o shn $(cli_args) .

.PHONY: build test format clean
