.PHONY: build test test-verbose test-cover clean lint

# Build the application
build:
	go build -o slack-scheduler .

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage report
test-cover:
	go test -cover ./...

# Generate HTML coverage report
test-cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	rm -f slack-scheduler coverage.out coverage.html

# Run go vet
vet:
	go vet ./...

# Format code
fmt:
	go fmt ./...

# Run all checks (format, vet, test)
check: fmt vet test
