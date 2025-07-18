// Package main provides a Kubernetes sidecar that performs leader election.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	filepath "path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// Default values for leader election timing
const (
	defaultLeaseDuration = 15 * time.Second
	defaultRenewDeadline = 10 * time.Second
	defaultRetryPeriod   = 2 * time.Second
)

// getEnvDuration gets duration from environment variable with LE_ prefix, returns default if not set
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv("LE_" + key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Invalid duration format for LE_%s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}

func getLeaseDirectoryPath() string {
	dirPath, exists := os.LookupEnv("LEASE_DIRECTORY")
	if !exists {
		dirPath = filepath.Join(os.TempDir(), "leader-elector")
	}

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Fatalf("Failed to create lease directory: %v", err)
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		log.Fatalf("Failed to stat lease directory: %v", err)
	}
	if !fileInfo.IsDir() {
		log.Fatalf("Lease directory path is not a directory: \"%s\". Use LEASE_DIRECTORY variable to set a proper directory.", dirPath)
	}

	return dirPath
}

// setupLeaderElection configures and returns the leader election configuration
func setupLeaderElection() (*leaderelection.LeaderElectionConfig, string, error) {
	// Get in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If in-cluster config fails, use KUBECONFIG
		kubeconfig, exists := os.LookupEnv("KUBECONFIG")
		if !exists {
			return nil, "", fmt.Errorf("failed to get in-cluster config and KUBECONFIG not set")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, "", fmt.Errorf("failed to build config from flags: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	id, err := os.Hostname()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get hostname: %w", err)
	}

	// Get lease and namespace from environment variables
	leaseName, exists := os.LookupEnv("LEASE_NAME")
	if !exists {
		leaseName = filepath.Base(os.Args[0])
	}

	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		log.Println("NAMESPACE environment variable is not set. Trying to fetch from config..")
		clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
		namespace = clientCfg.Contexts[clientCfg.CurrentContext].Namespace

		if namespace == "" {
			return nil, "", fmt.Errorf("NAMESPACE is not set")
		}
	}

	leaseDirectory := getLeaseDirectoryPath()
	leaderStatusFile := filepath.Join(leaseDirectory, leaseName)

	log.Printf("Starting leader election - LEASE_NAME: %s, NAMESPACE: %s, LEASE_DIRECTORY: %s", 
		leaseName, namespace, leaseDirectory)

	// Get timing values from environment variables
	leaseDuration := getEnvDuration("LEASE_DURATION", defaultLeaseDuration)
	renewDeadline := getEnvDuration("RENEW_DEADLINE", defaultRenewDeadline)
	retryPeriod := getEnvDuration("RETRY_PERIOD", defaultRetryPeriod)

	log.Printf("Leader election timing - LeaseDuration: %v, RenewDeadline: %v, RetryPeriod: %v",
		leaseDuration, renewDeadline, retryPeriod)

	// Lock required for leader election
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	// Create leader election configuration
	leaderElectionConfig := &leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDeadline,
		RetryPeriod:   retryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// we're now the leader
				log.Printf("Became leader for LEASE_NAME: %s in NAMESPACE: %s", 
					lock.LeaseMeta.Name, lock.LeaseMeta.Namespace)
				if err := updateStatus(id, leaderStatusFile); err != nil {
					log.Printf("Failed to update status file: %v", err)
				}
			},
			OnStoppedLeading: func() {
				// we are not the leader anymore
				log.Printf("Lost leadership for LEASE_NAME: %s in NAMESPACE: %s", 
					lock.LeaseMeta.Name, lock.LeaseMeta.Namespace)
				if err := removeStatusFile(leaderStatusFile); err != nil {
					log.Printf("Failed to remove status file: %v", err)
				}
			},
			OnNewLeader: func(identity string) {
				// we observe a new leader
				if identity != id {
					log.Printf("New leader elected: %s for LEASE_NAME: %s in NAMESPACE: %s", 
						identity, lock.LeaseMeta.Name, lock.LeaseMeta.Namespace)
					if err := removeStatusFile(leaderStatusFile); err != nil {
						log.Printf("Failed to remove status file: %v", err)
					}
				}
			},
		},
	}

	return leaderElectionConfig, leaderStatusFile, nil
}

func main() {
	leaderElectionConfig, _, err := setupLeaderElection()
	if err != nil {
		log.Fatalf("Failed to setup leader election: %v", err)
	}

	// Try and become the leader
	leaderelection.RunOrDie(context.Background(), *leaderElectionConfig)
}

func updateStatus(status string, leaderStatusFile string) error {
	f, err := os.Create(leaderStatusFile)
	if err != nil {
		return fmt.Errorf("failed to create status file: %w", err)
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Printf("Failed to close status file: %v", closeErr)
		}
	}()

	if _, err := f.WriteString(status); err != nil {
		return fmt.Errorf("failed to write status: %w", err)
	}
	
	if err := f.Sync(); err != nil {
		log.Printf("Failed to sync status file: %v", err)
	}
	
	return nil
}

func removeStatusFile(leaderStatusFile string) error {
	if err := os.Remove(leaderStatusFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove status file: %w", err)
	}
	return nil
}
