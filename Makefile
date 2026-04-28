.PHONY: run build test clean tidy

run:
	go run cmd/api/main.go

build:
	go build -o bin/bytevault.exe cmd/api/main.go

test:
	go test ./... -v

clean:
	del /Q bin\* 2>nul || true

tidy:
	go mod tidy