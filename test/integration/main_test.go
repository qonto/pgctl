package integration

import (
	"log"
	"os"
	"testing"
	"time"
)

// TestMain runs before all tests and provides setup/teardown for the entire test suite
func TestMain(m *testing.M) {
	// Log test environment info
	log.Println("Starting integration test suite...")
	log.Printf("Test timeout: %v", 10*time.Minute)

	_ = os.Setenv("TEST_ENV", "true")

	// Run all tests
	code := m.Run()

	// Cleanup
	log.Println("Integration test suite completed")

	os.Exit(code)
}
