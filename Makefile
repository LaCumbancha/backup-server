SHELL := /bin/bash
PYTHON := /usr/bin/python3.8
PWD := $(shell pwd)
GIT_REMOTE = github.com/LaCumbancha/backup-server

PROJECT_NAME = tp1

ECHOSV := 1
MANAGER := 1
BKP_MANAGERS := 1
ECHO_SERVERS := 2

NEW := 1

default: build

all:

deps:
	go mod tidy
	go mod vendor

build: deps
	GOOS=linux go build -o bin/manager $(GIT_REMOTE)/backup-manager
	GOOS=linux go build -o bin/echo-server $(GIT_REMOTE)/echo-server
.PHONY: build

docker-image:
	$(PYTHON) ./scripts/system-builder --bkp-managers=$(BKP_MANAGERS) --echo-servers=$(ECHO_SERVERS)
	docker build -f ./backup-manager/Dockerfile -t "bkp_manager:latest" .
	docker build -f ./echo-server/Dockerfile -t "echo_server:latest" .
.PHONY: docker-image

docker-compose-up: docker-image
	docker-compose -f docker-compose-dev.yaml --project-name $(PROJECT_NAME) up -d --build --remove-orphans
.PHONY: docker-compose-up

docker-compose-down:
	./scripts/stop-extra-services
	docker-compose -f docker-compose-dev.yaml --project-name $(PROJECT_NAME) stop -t 1
	docker-compose -f docker-compose-dev.yaml --project-name $(PROJECT_NAME) down
.PHONY: docker-compose-down

docker-compose-logs:
	docker-compose -f docker-compose-dev.yaml --project-name $(PROJECT_NAME) logs -f
.PHONY: docker-compose-logs

docker-manager-shell:
	docker container exec -it bkp_manager$(MANAGER) /bin/sh
.PHONY: docker-manager-shell

docker-echosv-shell:
	docker container exec -it echo_server$(ECHOSV) /bin/sh
.PHONY: docker-echosv-shell

docker-add-bkpmngr:
	$(eval START := $(shell ./scripts/next-service "bkp_manager"))
	$(eval END := $(shell echo $$(($(NEW) + $(START) - 1))))
	for idx in $(shell seq $(START) $(END)); do \
		docker run -d --rm \
		--name bkp_manager$$idx \
		--network=$(PROJECT_NAME)_testing_net \
		--mount type=bind,source=$(PWD)/bkp_manager/config,target=/config "bkp_manager:latest" \
		-c "export APP_CONFIG_FILE=/config/initial-config.yaml; ./bkp_manager"; \
	done
	./scripts/network-stats
.PHONY: docker-add-bkpmngr

docker-add-echosv:
	$(eval START := $(shell ./scripts/next-service "echo_server"))
	$(eval END := $(shell echo $$(($(NEW) + $(START) - 1))))
	for idx in $(shell seq $(START) $(END)); do \
		docker run -d --rm \
		--name echo_server$$idx \
		--network=$(PROJECT_NAME)_testing_net \
		--mount type=bind,source=$(PWD)/echo-server/config,target=/config "echo_server:latest" \
		-c "export APP_CONFIG_FILE=/config/initial-config.yaml; ./echo-server"; \
	done
	./scripts/network-stats
.PHONY: docker-add-echosv
