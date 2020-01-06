build: simplex

clean:
	touch simplex
	rm simplex

simplex:
	CGO_ENABLED=1 go build .

.PHONY: build clean
