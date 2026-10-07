.PHONY: run build test test-integration lint check check-structure dev-up dev-down migrate-up migrate-down migrate-status migration release

APP_NAME = apiserver
BUILD_DIR = ./build

run:
	go run .

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) .

# unit test — ห้ามแตะ DB/Redis
test:
	go test -race -count=1 ./...

# integration test — ต้อง make dev-up ก่อน (ใช้ Postgres/Redis จาก docker-compose)
test-integration:
	go test -race -count=1 -tags=integration ./...

lint:
	golangci-lint run ./...

check-structure:
	bash scripts/check_structure.sh

# รันก่อนเปิด PR ทุกครั้ง
check: check-structure
	go build ./...
	go vet ./...
	go test -count=1 ./...

# ---- local dependencies ----
dev-up:
	docker compose up -d

dev-down:
	docker compose down

# ---- migration (goose) ----
migrate-up:
	go run ./scripts/migrate up

migrate-down:
	go run ./scripts/migrate down

migrate-status:
	go run ./scripts/migrate status

# make migration name=agent_create_tables
migration:
	go run ./scripts/migrate create $(name)

# make superadmin username=admin  (พิมพ์รหัสผ่านตอนรัน)
superadmin:
	go run ./scripts/create_superadmin -username $(username)

# make admin username=support01  (พิมพ์รหัสผ่านตอนรัน)
admin:
	go run ./scripts/create_admin -username $(username)

# ---- release ----
# make release env=dev | make release env=prod — tag ที่ origin/<branch> แล้ว push ให้ CI build image (docs/DEPLOYMENT.md)
release:
	bash scripts/release_tag.sh $(env)
