SHELL := /bin/bash

.PHONY: docker-build
docker-build:
	docker build -t runabol/streamabol:latest .

.PHONY: docker-push
docker-push:
	@echo "$(DOCKER_PASSWORD)" | docker login -u $(DOCKER_LOGIN) --password-stdin
	docker push runabol/streamabol:latest
