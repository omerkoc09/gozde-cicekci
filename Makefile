.PHONY: db-up db-down test migrate-up migrate-down seed run

db-up:
	docker compose up -d
	@echo "Postgres hazır bekleniyor..."
	@until docker compose exec -T postgres pg_isready -U cicekci >/dev/null 2>&1; do sleep 1; done
	@until docker compose exec -T postgres_test pg_isready -U cicekci >/dev/null 2>&1; do sleep 1; done
	@echo "Hazır."

db-down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down 1

test:
	go test ./... -v

seed:
	go run ./cmd/seed

run:
	go run ./cmd/server
