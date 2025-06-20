#!/bin/sh

# todoer Test Runner
# This script runs different categories of tests for the todoer application
# Compatible with POSIX shells including fish, bash, and zsh

set -e

echo "🧪 todoer Test Suite Runner"
echo "=========================="

# Function to run tests with timing
run_test() {
    test_name="$1"
    test_cmd="$2"
    echo
    echo "📋 Running $test_name..."
    echo "Command: $test_cmd"
    time $test_cmd
}

# Parse command line arguments
case "${1:-all}" in
    "core")
        run_test "Core Package Tests" "go test ./pkg/core -v"
        ;;
    "integration")
        run_test "Integration Tests" "go test -run TestIntegration -v"
        ;;
    "cli")
        # Try to use timeout if available, otherwise run without timeout
        if command -v timeout >/dev/null 2>&1; then
            run_test "CLI Tests (with timeout)" "timeout 60s go test -run TestCLICommands -v"
        else
            run_test "CLI Tests" "go test -run TestCLICommands -v"
        fi
        ;;
    "basic")
        run_test "Basic Functionality Tests" "go test -run 'TestExtractDateFromFrontmatter|TestGeneratorLibraryInterface' -v"
        ;;
    "quick")
        echo "🚀 Running Quick Test Suite..."
        run_test "Core Tests" "go test ./pkg/core -short"
        run_test "Basic Integration" "go test -run TestIntegration"
        run_test "Generator Tests" "go test -run TestGeneratorLibraryInterface"
        ;;
    "all")
        echo "🎯 Running Full Test Suite..."
        run_test "Core Package Tests" "go test ./pkg/core -v"
        run_test "Integration Tests" "go test -run TestIntegration -v"
        run_test "Generator Tests" "go test -run TestGeneratorLibraryInterface -v"
        run_test "Basic CLI Tests" "go test -run TestBasicCLIFunctionality -v"
        echo
        echo "ℹ️  Note: Full CLI tests (TestCLICommands) can be run with: $0 cli"
        ;;
    "help")
        echo "Usage: $0 [test_type]"
        echo
        echo "Available test types:"
        echo "  all         - Run all tests (except long-running CLI tests)"
        echo "  core        - Run core package unit tests"
        echo "  integration - Run integration tests"
        echo "  cli         - Run full CLI integration tests (may take longer)"
        echo "  basic       - Run basic functionality tests"
        echo "  quick       - Run quick smoke tests"
        echo "  help        - Show this help message"
        echo
        echo "Examples:"
        echo "  $0              # Run all tests"
        echo "  $0 core         # Run just core tests"
        echo "  $0 quick        # Run quick smoke tests"
        exit 0
        ;;
    *)
        echo "❌ Unknown test type: $1"
        echo "Run '$0 help' for available options"
        exit 1
        ;;
esac

echo
echo "✅ Test run completed!"
