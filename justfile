# Build the ytx CLI binary
build:
    @echo "Building ytx binary..."
    go build -o ./tmp/ytx ./cmd/main.go
    @echo "Built: ./tmp/ytx"

# Run all tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with coverage report
cover:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=./tmp/coverage.out ./...
    go tool cover -html=./tmp/coverage.out -o ./tmp/coverage.html
    @echo "Coverage report generated: ./tmp/coverage.html"

# Python coverage
coverage:
    @echo "Switching to FastAPI proxy project"
    source music/.venv/bin/activate
    coverage run -m pytest music/tests/
    @coverage html
