## help: print this help message
.PHONY: help
help:
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: build your program
.PHONY: build
build:
	@go build -o bin/gobank

## run: execute binary saved on bin folder 
.PHONY: run
run: build
	@./bin/gobank

## test: run tests
.PHONY: test
test:
	@go test -v ./...

## clean: remove bin folder
.PHONY: clean
clean:
	@rm -fr bin/
