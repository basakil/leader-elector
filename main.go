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

// string leaderStatusFile = os.TempDir() + "/leader-elector"

func getLeaseDirectoryPath() string {
	dirPath, exists := os.LookupEnv("LEASE_DIRECTORY")
	if !exists {
		dirPath = filepath.Join(os.TempDir(), "leader-elector")
	}

	if err := os.Mkdir(dirPath, os.ModeDir); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		log.Fatal(err)
	}
	if !fileInfo.IsDir() {
		log.Fatal("Lease directory path is not a directory: \"" + dirPath + "\" . Use LEASE_DIRECTORY variable to set a proper (existing or new) directory.")
	}

	return dirPath
}

func main() {
	// Get in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If in-cluster config fails, use KUBECONFIG
		kubeconfig, exists := os.LookupEnv("KUBECONFIG")
		if !exists {
			log.Fatal("Failed to get in-cluster config and KUBECONFIG not set")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	id, err := os.Hostname()
	if err != nil {
		log.Fatal(err.Error())
	}

	// Get lease and namespace from environment variables
	leaseName, exists := os.LookupEnv("LEASE_NAME")
	if !exists {
		leaseName = filepath.Base(os.Args[0])
		// panic("LEASE_NAME not set")
	}

	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		log.Println("NAMESPACE environment variable is not set. Trying to fetch from config..")
		clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
		namespace = clientCfg.Contexts[clientCfg.CurrentContext].Namespace

		if namespace == "" {
			log.Fatal("NAMESPACE is not set")
		}
	}

	leaseDirectory := getLeaseDirectoryPath()
	leaderStatusFile := filepath.Join(leaseDirectory, leaseName)

	log.Println("Using LEASE_NAME: ", leaseName, "in NAMESPACE: ", namespace, " . LEASE_DIRECTORY: ", leaseDirectory)

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

	// Try and become the leader
	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// we're now the leader
				log.Println("This application is the leader for LEASE_NAME: " + lock.LeaseMeta.Name + " in NAMESPACE: " + lock.LeaseMeta.Namespace + " .")
				updateStatus(id, leaderStatusFile)
			},
			OnStoppedLeading: func() {
				// we are not the leader anymore
				log.Println("This application has lost leadership for LEASE_NAME: " + lock.LeaseMeta.Name + " in NAMESPACE: " + lock.LeaseMeta.Namespace + " .")
				removeStatusFile(leaderStatusFile)
			},
			OnNewLeader: func(identity string) {
				// we observe a new leader
				if identity != id {
					log.Println("This application has lost leadership for LEASE_NAME: " + lock.LeaseMeta.Name + " in NAMESPACE: " + lock.LeaseMeta.Namespace + " to " + id + " .")
					removeStatusFile(leaderStatusFile)
				}
			},
		},
	})
}

func updateStatus(status string, leaderStatusFile string) {
	f, err := os.Create(leaderStatusFile)
	if err != nil {
		panic(err.Error())
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("failed to close file: %v", err)
		}
	}()

	_, err = f.WriteString(status)
	if err != nil {
		panic(err.Error())
	}
	if err := f.Sync(); err != nil {
		fmt.Printf("failed to sync file: %v", err)
	}
}

func removeStatusFile(leaderStatusFile string) {
	if err := os.Remove(leaderStatusFile); err != nil {
		fmt.Printf("failed to remove %s, error: %s", leaderStatusFile, err)
	}
}
