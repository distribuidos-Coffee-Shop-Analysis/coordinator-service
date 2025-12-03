SHELL := /bin/bash
PWD := $(shell pwd)

GIT_REMOTE = github.com/7574-sistemas-distribuidos/docker-compose-init

all:

docker-compose-up:
	docker compose up --build
.PHONY: docker-compose-up

docker-compose-down:
	docker compose down -v
.PHONY: docker-compose-down
