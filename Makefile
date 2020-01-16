test_target ?= ./...

build: simplex

format:
	gofmt -s -w .

test:
	go test $(cli_args) $(test_target)

clean:
	touch simplex
	rm simplex

simplex:
	CGO_ENABLED=1 go build $(cli_args) .

.PHONY: build test format clean
