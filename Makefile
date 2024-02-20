usage: FORCE
	exit 1

FORCE:

include config.env
export $(shell sed 's/=.*//' config.env)

build: FORCE
	@echo " >> building..."
	@go build

start: FORCE
	@echo " >> building..."
	@mkdir -p log
	@go build
	@./grpc-webui

test: FORCE
	@echo " >> testing..."
	@go test -v ./...

clean: FORCE
	@echo " >> cleaning..."
	@rm -rf log
	@rm -f grpc-webui

.PHONY: start test clean
