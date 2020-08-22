build-github-token-sidecar:
	CGO_ENABLED=0 go build -o bin/github-token-sidecar cmd/github-token-sidecar/main.go

build-github-token-sidecar-image: build-github-token-sidecar
	docker build -f Dockerfile.github-token-sidecar .

fmt:
	go fmt ./...