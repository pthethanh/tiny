PROJECT_NAME=tiny
GO_BUILD_ENV=CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on
GO_FILES=$(shell go list ./... | grep -v /vendor/)
GOPATH ?= $(HOME)/go

export PATH := $(GOPATH)/bin:$(PATH)

.SILENT:

all: mod_download fmt vet build test

build_test: fmt vet build test

vet:
	$(GO_BUILD_ENV) go vet $(GO_FILES)

fmt:
	$(GO_BUILD_ENV) go fmt $(GO_FILES)

test:
	$(GO_BUILD_ENV) go test $(GO_FILES) -cover -v

mod_download:
	$(GO_BUILD_ENV) go mod download

build: mod_download
	$(GO_BUILD_ENV) go build -v .
