package olareg

import (
	"github.com/testcontainers/testcontainers-go"
)

const (
	containerRegistryPath string = "/home/appuser/registry"
)

// WithData is used to initialize the repository with a directory of OCI Layouts.
// The content will not be modified.
func WithData(dataPath string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Files = append(req.Files, testcontainers.ContainerFile{
			HostFilePath:      dataPath,
			ContainerFilePath: containerRegistryPath,
		})
		return nil
	}
}
