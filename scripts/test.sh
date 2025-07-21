#!/bin/bash

# Test script for leader-elector project

set -e

# Change to project root (parent of script directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/.."
cd "$PROJECT_ROOT"

echo "Running leader-elector tests..."

# Function to run tests with different options
run_tests() {
    echo "Running basic tests..."
    go test -v ./...
}

run_coverage() {
    echo "Running tests with coverage..."
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
    echo "Coverage report generated: coverage.out"
}

run_benchmarks() {
    echo "Running benchmarks..."
    go test -bench=. -benchmem ./...
}

run_race_detection() {
    echo "Running tests with race detection..."
    go test -race ./...
}

run_all() {
    echo "Running all tests..."
    run_tests
    echo ""
    run_coverage
    echo ""
    run_benchmarks
    echo ""
    run_race_detection
}

# Parse command line arguments
case "${1:-all}" in
    "tests")
        run_tests
        ;;
    "coverage")
        run_coverage
        ;;
    "benchmarks")
        run_benchmarks
        ;;
    "race")
        run_race_detection
        ;;
    "all")
        run_all
        ;;
    *)
        echo "Usage: $0 [tests|coverage|benchmarks|race|all]"
        echo "  tests      - Run basic tests"
        echo "  coverage   - Run tests with coverage report"
        echo "  benchmarks - Run performance benchmarks"
        echo "  race       - Run tests with race detection"
        echo "  all        - Run all tests (default)"
        exit 1
        ;;
esac

echo "Tests completed successfully!" 