.PHONY: test test-coverage test-integration clean-integration coverage clean

test:
	go test -race -v ./...

test-coverage:
	go test -race -coverprofile=coverage.out -v ./...
	go tool cover -func=coverage.out

test-integration:
	docker compose -f docker-compose.integration.yml up -d --wait
	@echo "Waiting for services to be ready..."
	@sleep 5
	go test -tags=integration -coverprofile=integration-coverage.out -timeout 120s -v -run=Integration ./...
	@echo "Integration tests completed. Run 'make clean-integration' to tear down containers."

coverage: test-coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

clean-integration:
	docker compose -f docker-compose.integration.yml down -v

clean:
	rm -f coverage.out integration-coverage.out coverage.html