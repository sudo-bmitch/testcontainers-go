package olareg_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/olareg"
)

func TestRegistry_ping(t *testing.T) {
	container, err := olareg.RunContainer(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Clean up the container after the test is complete
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	httpAddress, err := container.Address(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(httpAddress + "/v2/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200, but got %d", resp.StatusCode)
	}
}

func TestRegistry_data(t *testing.T) {
	ctx := context.Background()
	container, err := olareg.RunContainer(ctx, olareg.WithData("testdata/registry"))
	if err != nil {
		t.Fatal(err)
	}

	// Clean up the container after the test is complete
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	httpAddress, err := container.Address(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(httpAddress + "/v2/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200, but got %d", resp.StatusCode)
	}

	err = container.ImageExists(ctx, container.RegistryName+"/busybox:latest")
	if err != nil {
		t.Errorf("failed to query for image: %v", err)
	}

}
