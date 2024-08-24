package postgrestest

import (
	"context"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SpinUpContainer(ctx context.Context, user, password, defaultDBName string) (addr string, terminate func(context.Context) error, err error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "postgres:16.3",
			Env: map[string]string{
				"POSTGRES_USER":     user,
				"POSTGRES_PASSWORD": password,
				"POSTGRES_DB":       defaultDBName,
			},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor:   wait.ForLog("database system is ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		return "", nil, err
	}
	addr, err = container.Endpoint(ctx, "")
	if err != nil {
		return "", nil, err
	}
	terminate = func(ctx context.Context) error { return container.Terminate(ctx) }
	// Poll container health status until it's healthy.
	interval := 1 * time.Millisecond
	for {
		select {
		case <-time.After(interval):
			isHealthy, err := healthCheckDB(addr, user, password, defaultDBName)
			if err != nil {
				return "", nil, err
			}
			if !isHealthy {
				interval *= 2 // Exponential backoff.
				continue
			}
			return addr, terminate, nil
		case <-ctx.Done():
			return "", nil, ctx.Err()
		}
	}
}
