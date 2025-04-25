.DEFAULT_GOAL := start

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet: fmt
	go vet ./...

.PHONY: build
build: vet
	go build -o archaide-server ./cmd/archaide/main.go

.PHONY: start
start: build
	./archaide-server

.PHONY: dev
dev:
	go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "./archaide-server" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go,tpl,tmpl,html,css,scss,js,ts,sql,jpeg,jpg,gif,png,bmp,svg,webp,ico" \
		--misc.clean_on_exit "true"

.PHONY: clean
clean:
	rm -rf ./archaide-server
