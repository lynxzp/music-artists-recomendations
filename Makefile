IMAGE    := lynxzp/music-recomendations
PLATFORM := linux/amd64

.DEFAULT_GOAL := all

test:
	go test -race ./...

build: test
	docker build --platform $(PLATFORM) -t $(IMAGE) .

push: build
	docker push $(IMAGE)

all: push

.PHONY: test build push all
