.PHONY: help tidy build run test test-unit test-integration fmt vet lint clean

BINARY := bin/penbun-api
PKG    := ./...

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

tidy: ## ดาวน์โหลดและจัดระเบียบ dependency
	go mod tidy

fmt: ## จัดรูปแบบโค้ดทั้งโปรเจกต์
	gofmt -w .

vet: ## ตรวจโค้ดด้วย go vet
	go vet $(PKG)

build: ## คอมไพล์เป็นไฟล์เดียว
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) .

run: ## รันบนเครื่องตัวเอง
	go run .

test-unit: ## รันเฉพาะเทสต์ที่ไม่ต้องใช้ฐานข้อมูล
	go test -race -short $(PKG)

test-integration: ## รันเทสต์ที่ต้องใช้ฐานข้อมูลจริง
	go test -race -tags=integration ./test/integration

test: test-unit ## ค่าเริ่มต้นคือเทสต์ที่ไม่ต้องใช้ฐานข้อมูล

clean:
	rm -rf bin/
