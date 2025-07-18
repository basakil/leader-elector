# leader-elector

This project provides a simple sidecar container that performs leader election in a Kubernetes cluster. The sidecar uses the Kubernetes client-go library to create a Lease object in the Kubernetes API, allowing multiple replicas of your application to decide which is the leader.

## How it Works

The sidecar uses the Kubernetes leader election functionality, built around a Lease object. The Lease object is a Kubernetes primitive that represents a distributed lock. When multiple replicas of your application run, they will compete to acquire the lock. The one that succeeds becomes the leader.

The leader writes their identity to a file (`/tmp/leader_status`) to indicate that it's the leader. If it loses leadership, the file is deleted.

If the leader crashes or stops renewing the lease, another replica will acquire the lease and become the new leader.

The identity used for each replica is the pod's hostname.

## Features

- **Configurable timing**: All leader election timing parameters can be configured via environment variables
- **Logback-style logging**: Consistent logging format for easy aggregation
- **Graceful error handling**: Proper error handling without panics
- **Comprehensive testing**: Full unit test coverage with benchmarks

## Environment Variables

### Required
- `LEASE_NAME`: Name of the lease object (defaults to executable name)
- `NAMESPACE`: Kubernetes namespace for the lease (auto-detected if not set)

### Optional
- `LEASE_DIRECTORY`: Directory for status files (defaults to `/tmp/leader-elector`)
- `LE_LEASE_DURATION`: Duration of the lease (default: `15s`)
- `LE_RENEW_DEADLINE`: Deadline for renewing the lease (default: `10s`)
- `LE_RETRY_PERIOD`: Period between retry attempts (default: `2s`)

## Deployment

To use the sidecar in your Kubernetes deployment, add it to the list of containers in your pod specification:

```yaml
spec:
  containers:
  - name: myapp
    image: myapp:1.0.0
  - name: leader-election-sidecar
    image: supporttools/leader-elector:latest
    env:
    - name: LEASE_NAME
      value: myapp-lock
    - name: NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    - name: LE_LEASE_DURATION
      value: "30s"
    - name: LE_RENEW_DEADLINE
      value: "20s"
    - name: LE_RETRY_PERIOD
      value: "5s"
    volumeMounts:
    - name: leader-status
      mountPath: /tmp/leader_status
  volumes:
  - name: leader-status
    emptyDir: {}
```

In this configuration, `myapp` is the main container of the pod, and `leader-election-sidecar` is the sidecar container that performs leader election. The `LEASE_NAME` and `NAMESPACE` environment variables specify the name of the Lease object and the namespace where it's created. The `NAMESPACE` is set from the pod's metadata, automatically matching the namespace where the pod is running.

The sidecar shares an `emptyDir` volume with the main container, where it writes the leader status file. Your application can watch this file to know if it's the leader. The leader file will only exist on the leader pod. Also, the leader_status file has the hostname of the leader pod inside it.

Here is an example of using this in your application.

```bash
#!/bin/bash

while true
do
  echo "Checking leadership."
  if [ -f /tmp/leader_status ]
  then
    echo "I am the leader!!!"
    # Start your application here
  else
    echo "I am not the leader, sleeping..."
    sleep 5
  fi
done
```

## Building and Running Locally

You can build and run this project locally with Go:

```bash
go build -o leader-election-sidecar main.go
LEASE_NAME=myapp-lock NAMESPACE=default ./leader-election-sidecar
```

This will run the sidecar and attempt to perform leader election using your local Kubernetes context (either from your in-cluster configuration or from `$KUBECONFIG`).

## Testing

This project includes comprehensive unit tests and benchmarks. To run the tests:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run specific test
go test -run TestGetEnvDuration ./...

# Run tests with race detection
go test -race ./...
```

### Test Coverage

The test suite covers:

- **Environment variable parsing**: Testing all timing configuration scenarios
- **File operations**: Status file creation, reading, and deletion
- **Error handling**: Invalid inputs and edge cases
- **Leader election callbacks**: All callback functions
- **Concurrency**: Concurrent access to status files
- **Benchmarks**: Performance testing for critical functions

### Test Files

- `main_test.go`: Basic unit tests for core functionality
- `leader_election_test.go`: Advanced tests with Kubernetes client mocking

### Running Tests in CI/CD

For continuous integration, you can run:

```bash
# Install dependencies
go mod download

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Generate coverage report
go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## Security

This project has been updated to use the latest secure versions:

- **Go 1.24.5**: Latest stable version with security patches
- **Kubernetes client-go v0.33.3**: Latest stable version
- **Alpine Linux**: Minimal base image for reduced attack surface
- **CA certificates**: Included for secure HTTPS connections

## Contributing

When contributing to this project:

1. Write tests for new functionality
2. Ensure all tests pass
3. Update documentation as needed
4. Follow Go best practices and conventions
