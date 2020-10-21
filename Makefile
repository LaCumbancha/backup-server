SHELL := /bin/bash
PWD := $(shell pwd)
ID := 1

GIT_REMOTE = github.com/LaCumbancha/backup-server

default: build

all:

deps:
	go mod tidy
	go mod vendor

build: deps
	GOOS=linux go build -o bin/manager github.com/LaCumbancha/backup-server/backup-manager
.PHONY: build

docker-image:
	docker build -f ./backup-manager/Dockerfile -t "bkpManager:latest" .
.PHONY: docker-image

docker-compose-up: docker-image
	docker-compose -f docker-compose-dev.yaml up -d --build
.PHONY: docker-compose-up

docker-compose-down:
	docker-compose -f docker-compose-dev.yaml stop -t 1
	docker-compose -f docker-compose-dev.yaml down
.PHONY: docker-compose-down

docker-compose-logs:
	docker-compose -f docker-compose-dev.yaml logs -f
.PHONY: docker-compose-logs

docker-bkpManager-shell:
	docker container exec -it bkpManager /bin/bash
.PHONY: docker-server-shell
