#!/bin/bash

# Test script for leader-elector project

set -e

print_help() {
  cat <<EOF
Run tests, coverage, benchmarks, or race detection for the leader-elector Go project.

Usage:
  $0 [tests|coverage|benchmarks|race|all]

Parameters:
  tests       Run basic tests
  coverage    Run tests with coverage report
  benchmarks  Run performance benchmarks
  race        Run tests with race detection
  all         Run all tests (default)
  -h, --help  Show this help message and exit

Examples:
  $0 tests
  $0 coverage
  $0 all
EOF
}

# Find project root (parent of script directory) without cd
SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")
PROJECT_ROOT="$SCRIPTPATH/.."

echo "Running leader-elector tests..."

# Function to run tests with different options
run_tests() {
    echo "Running basic tests..."
    (cd "$PROJECT_ROOT" && go test -v ./...)
}

run_coverage() {
    echo "Running tests with coverage..."
    (cd "$PROJECT_ROOT" && go test -coverprofile=coverage.out ./...)
    (cd "$PROJECT_ROOT" && go tool cover -func=coverage.out)
    echo "Coverage report generated: $PROJECT_ROOT/coverage.out"
}

run_benchmarks() {
    echo "Running benchmarks..."
    (cd "$PROJECT_ROOT" && go test -bench=. -benchmem ./...)
}

run_race_detection() {
    echo "Running tests with race detection..."
    (cd "$PROJECT_ROOT" && go test -race ./...)
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
    "-h"|"--help")
        print_help
        exit 0
        ;;
    *)
        print_help
        exit 1
        ;;
esac

echo "Tests completed successfully!" 