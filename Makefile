postgres:
	docker run --name mock_bank -e POSTGRES_PASSWORD=secret -e POSTGRES_USER=root -p 5432:5432 -d postgres:alpine

createdb:
	docker exec -it mock_bank createdb --username=root --owner=root mock_bank

dropdb:
	docker exec -it mock_bank dropdb mock_bank

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/mock_bank?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/mock_bank?sslmode=disable" -verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server