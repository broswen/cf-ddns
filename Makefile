
.PHONY: build
build:
	CGO_ENABLED=0 go build

.PHONY: docker
docker:
	docker build . -t cf-ddns