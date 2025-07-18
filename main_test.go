package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "valid duration",
			envKey:       "LEASE_DURATION",
			envValue:     "30s",
			defaultValue: 15 * time.Second,
			expected:     30 * time.Second,
		},
		{
			name:         "valid duration with minutes",
			envKey:       "RENEW_DEADLINE",
			envValue:     "2m",
			defaultValue: 10 * time.Second,
			expected:     2 * time.Minute,
		},
		{
			name:         "invalid duration",
			envKey:       "RETRY_PERIOD",
			envValue:     "invalid",
			defaultValue: 2 * time.Second,
			expected:     2 * time.Second,
		},
		{
			name:         "empty environment variable",
			envKey:       "LEASE_DURATION",
			envValue:     "",
			defaultValue: 15 * time.Second,
			expected:     15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("LE_"+tt.envKey, tt.envValue)
				defer os.Unsetenv("LE_" + tt.envKey)
			}

			result := getEnvDuration(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetLeaseDirectoryPath(t *testing.T) {
	tests := []struct {
		name           string
		leaseDirEnv    string
		shouldCreate   bool
		expectedPrefix string
		expectError    bool
	}{
		{
			name:           "default temp directory",
			leaseDirEnv:    "",
			shouldCreate:   true,
			expectedPrefix: os.TempDir(),
			expectError:    false,
		},
		{
			name:           "custom directory from env",
			leaseDirEnv:    "/tmp/custom-lease",
			shouldCreate:   true,
			expectedPrefix: "/tmp/custom-lease",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if provided
			if tt.leaseDirEnv != "" {
				os.Setenv("LEASE_DIRECTORY", tt.leaseDirEnv)
				defer os.Unsetenv("LEASE_DIRECTORY")
			}

			// Create a temporary directory for testing
			tempDir := t.TempDir()
			if tt.leaseDirEnv == "" {
				os.Setenv("LEASE_DIRECTORY", tempDir)
				defer os.Unsetenv("LEASE_DIRECTORY")
			}

			// Test the function
			result := getLeaseDirectoryPath()

			// Verify the result
			if tt.expectError {
				// This test case would require mocking os.Stat to return an error
				// For now, we'll skip error cases as they call log.Fatalf
				t.Skip("Error cases require mocking and would call log.Fatalf")
			}

			if tt.leaseDirEnv != "" && result != tt.leaseDirEnv {
				t.Errorf("getLeaseDirectoryPath() = %v, want %v", result, tt.leaseDirEnv)
			}

			// Verify the directory exists
			if _, err := os.Stat(result); os.IsNotExist(err) {
				t.Errorf("getLeaseDirectoryPath() returned path that doesn't exist: %v", result)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		leaderStatusFile string
		expectError    bool
	}{
		{
			name:           "valid status update",
			status:         "test-hostname",
			leaderStatusFile: filepath.Join(t.TempDir(), "test-status"),
			expectError:    false,
		},
		{
			name:           "empty status",
			status:         "",
			leaderStatusFile: filepath.Join(t.TempDir(), "test-status"),
			expectError:    false,
		},
		{
			name:           "invalid directory",
			status:         "test-hostname",
			leaderStatusFile: "/nonexistent/path/status",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateStatus(tt.status, tt.leaderStatusFile)

			if tt.expectError && err == nil {
				t.Errorf("updateStatus() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("updateStatus() unexpected error: %v", err)
			}

			// Verify file was created and contains correct content
			if !tt.expectError {
				content, readErr := os.ReadFile(tt.leaderStatusFile)
				if readErr != nil {
					t.Errorf("Failed to read status file: %v", readErr)
				}
				if string(content) != tt.status {
					t.Errorf("Status file content = %s, want %s", string(content), tt.status)
				}
			}
		})
	}
}

func TestRemoveStatusFile(t *testing.T) {
	tests := []struct {
		name           string
		leaderStatusFile string
		createFile     bool
		expectError    bool
	}{
		{
			name:           "remove existing file",
			leaderStatusFile: filepath.Join(t.TempDir(), "test-status"),
			createFile:     true,
			expectError:    false,
		},
		{
			name:           "remove non-existent file",
			leaderStatusFile: filepath.Join(t.TempDir(), "nonexistent-status"),
			createFile:     false,
			expectError:    false, // Should not error for non-existent files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create file if needed
			if tt.createFile {
				err := os.WriteFile(tt.leaderStatusFile, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			err := removeStatusFile(tt.leaderStatusFile)

			if tt.expectError && err == nil {
				t.Errorf("removeStatusFile() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("removeStatusFile() unexpected error: %v", err)
			}

			// Verify file was removed
			if tt.createFile {
				if _, statErr := os.Stat(tt.leaderStatusFile); !os.IsNotExist(statErr) {
					t.Errorf("Status file still exists after removal")
				}
			}
		})
	}
}

func TestSetupLeaderElection(t *testing.T) {
	// This test requires a Kubernetes cluster or mocked client
	// For now, we'll test the error cases that don't require a cluster
	t.Run("missing KUBECONFIG", func(t *testing.T) {
		// Ensure KUBECONFIG is not set
		os.Unsetenv("KUBECONFIG")
		
		// Mock rest.InClusterConfig to return error
		// This would require dependency injection or mocking
		// For now, we'll skip this test as it requires complex setup
		t.Skip("Requires mocking Kubernetes client or running in cluster")
	})
}

// Benchmark tests
func BenchmarkGetEnvDuration(b *testing.B) {
	os.Setenv("LE_LEASE_DURATION", "30s")
	defer os.Unsetenv("LE_LEASE_DURATION")

	for i := 0; i < b.N; i++ {
		getEnvDuration("LEASE_DURATION", 15*time.Second)
	}
}

func BenchmarkUpdateStatus(b *testing.B) {
	tempDir := b.TempDir()
	statusFile := filepath.Join(tempDir, "bench-status")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updateStatus("test-hostname", statusFile)
	}
}

// Integration test helper
func TestIntegration(t *testing.T) {
	// This would be a full integration test that requires a Kubernetes cluster
	// For now, we'll skip it
	t.Skip("Integration test requires Kubernetes cluster")
}

// Test constants
func TestConstants(t *testing.T) {
	if defaultLeaseDuration != 15*time.Second {
		t.Errorf("defaultLeaseDuration = %v, want %v", defaultLeaseDuration, 15*time.Second)
	}
	if defaultRenewDeadline != 10*time.Second {
		t.Errorf("defaultRenewDeadline = %v, want %v", defaultRenewDeadline, 10*time.Second)
	}
	if defaultRetryPeriod != 2*time.Second {
		t.Errorf("defaultRetryPeriod = %v, want %v", defaultRetryPeriod, 2*time.Second)
	}
} 