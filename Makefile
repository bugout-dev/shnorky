build: simplex

test:
	go test ./... -v

clean:
	touch simplex
	rm simplex

simplex:
	CGO_ENABLED=1 go build .

.PHONY: build test clean
