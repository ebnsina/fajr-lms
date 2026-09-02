.PHONY: up down run test lint migrate migrate-down sqlc tidy web web-check web-build

up:        ; docker compose up -d --wait
down:      ; docker compose down
run:       ; go run ./cmd/api
test:      ; go test ./... -race -count=1
lint:      ; gofmt -l . && go vet ./...
tidy:      ; go mod tidy
migrate:   ; go run ./cmd/migrate up
migrate-down: ; go run ./cmd/migrate down
sqlc:      ; go tool sqlc generate
web:       ; cd web && npm run dev
web-check: ; cd web && npx svelte-check --tsconfig ./tsconfig.json
web-build: ; cd web && npm run build
