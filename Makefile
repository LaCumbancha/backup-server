SHELL := /bin/bash
PWD := $(shell pwd)
ID := 1

GIT_REMOTE = github.com/7574-sistemas-distribuidos/docker-compose-init

default: build

all:

deps:
	go mod tidy
	go mod vendor

build: deps
	GOOS=linux go build -o bin/client github.com/LaCumbancha/docker-init/client
	GOOS=linux go build -o bin/server github.com/LaCumbancha/docker-init/server
.PHONY: build

docker-image:
	docker build -f ./server/Dockerfile -t "server:latest" .
	docker build -f ./client/Dockerfile -t "client:latest" .
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

docker-server-shell:
	docker container exec -it server /bin/bash
.PHONY: docker-server-shell

docker-client-shell:
	docker container exec -it client$(ID) /bin/sh
.PHONY: docker-client-shell
