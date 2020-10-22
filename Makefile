SHELL := /bin/bash
PWD := $(shell pwd)
ECHOSV := 1

GIT_REMOTE = github.com/LaCumbancha/backup-server

default: build

all:

deps:
	go mod tidy
	go mod vendor

build: deps
	GOOS=linux go build -o bin/manager github.com/LaCumbancha/backup-server/backup-manager
	GOOS=linux go build -o bin/echo-server github.com/LaCumbancha/backup-server/echo-server
.PHONY: build

docker-image:
	docker build -f ./backup-manager/Dockerfile -t "bkp_manager:latest" .
	docker build -f ./echo-server/Dockerfile -t "echo_server:latest" .
.PHONY: docker-image

docker-compose-up: docker-image
	docker-compose -f docker-compose-dev.yaml up -d --build --remove-orphans
.PHONY: docker-compose-up

docker-compose-down:
	docker-compose -f docker-compose-dev.yaml stop -t 1
	docker-compose -f docker-compose-dev.yaml down
.PHONY: docker-compose-down

docker-compose-logs:
	docker-compose -f docker-compose-dev.yaml logs -f
.PHONY: docker-compose-logs

docker-manager-shell:
	docker container exec -it bkp_manager /bin/sh
.PHONY: docker-manager-shell

docker-echosv-shell:
	docker container exec -it echo_server$(ECHOSV) /bin/sh
.PHONY: docker-manager-shell
