DEFAULT: goimports

.PHONY: goimports
goimports: fmt
	gosimports -l -w .

.PHONY: fmt
fmt:
	find . -name '*.go' | grep -v vendor | xargs gofmt -s -w

.PHONY: run
run:
	go run .
