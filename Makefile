GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto | grep -v xviz ")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api/xviz -name *.proto ")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto | grep -v xviz )
	XVIZ_PROTO_FILES=$(shell find api/xviz -name *.proto )
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.1
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/go-kratos/kratos/cmd/kratos/v2@v2.0.0-20240322155018-41971ffa647a
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.0.0-20240322155018-41971ffa647a
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0
	go install github.com/google/wire/cmd/wire@v0.6.0
	#go install github.com/grpc-ecosystem/grpc-gateway/v2@v2.22.0
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@v2.0.0-20240322155018-41971ffa647a
	go install github.com/envoyproxy/protoc-gen-validate@v1.1.0
.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: xviz
# generate internal proto
xviz:
	protoc --proto_path=./api \
           --proto_path=./third_party \
           --go_out=paths=source_relative:./api_gen \
           $(XVIZ_PROTO_FILES)

.PHONY: clean
# clean generate code
clean:
	rm -rf ./api_gen && rm -f ./internal/conf/conf.pb.go

.PHONY: api
# generate api proto
api:
	rm -rf ./api_gen
	mkdir ./api_gen
	make xviz;
	protoc --proto_path=./api \
	       --proto_path=./third_party \
		   --go_out=paths=source_relative:./api_gen \
 	       --go-http_out=paths=source_relative:./api_gen \
 	       --go-grpc_out=paths=source_relative:./api_gen \
 	       --openapiv2_out=./api_gen  \
		   --openapiv2_opt logtostderr=true \
		   --go-errors_out=paths=source_relative:./api_gen \
		   --validate_out=paths=source_relative,lang=go:./api_gen \
	       $(API_PROTO_FILES)

.PHONY: build
# build
build:
#	make init
#	make api
#	make xviz
#	make config
#	make generate
	mkdir -p bin/ &&  go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
generate:
	go mod tidy
	#go get github.com/google/wire/cmd/wire@v0.6.0
	go generate ./...

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;


# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
