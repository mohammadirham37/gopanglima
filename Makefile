.PHONY: all fmt fmt-check vet test race cover bench check

all: check

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -cover ./...

bench:
	go test -run '^$$' -bench . -benchmem ./framework/...

check: fmt-check vet race
