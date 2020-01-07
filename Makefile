build: simplex

format:
	gofmt -s -w .

test:
	go test ./...

clean:
	touch simplex
	rm simplex

simplex:
	CGO_ENABLED=1 go build .

.PHONY: build test format clean
