package olareg

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	olaregImage = "ghcr.io/olareg/olareg:latest"
	olaregPort  = "5000/tcp"
)

// RegistryContainer represents the Registry container type used in the module
type RegistryContainer struct {
	testcontainers.Container
	RegistryName string
}

// Address returns the address of the Registry container, using the HTTP protocol
func (c *RegistryContainer) Address(ctx context.Context) (string, error) {
	port, err := c.MappedPort(ctx, "5000")
	if err != nil {
		return "", err
	}

	ipAddress, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%s", ipAddress, port.Port()), nil
}

var (
	hostPartS   = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)`
	hostPortS   = `(?:` + hostPartS + `(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)*` + regexp.QuoteMeta(`.`) + `?` + regexp.QuoteMeta(`:`) + `[0-9]+)`
	hostDomainS = `(?:` + hostPartS + `(?:(?:` + regexp.QuoteMeta(`.`) + hostPartS + `)+` + regexp.QuoteMeta(`.`) + `?|` + regexp.QuoteMeta(`.`) + `))`
	hostUpperS  = `(?:[a-zA-Z0-9]*[A-Z][a-zA-Z0-9-]*[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[A-Z][a-zA-Z0-9]*)`
	registryS   = `(?:` + hostDomainS + `|` + hostPortS + `|` + hostUpperS + `|localhost(?:` + regexp.QuoteMeta(`:`) + `[0-9]+)?)`
	repoPartS   = `[a-z0-9]+(?:(?:\.|_|__|-+)[a-z0-9]+)*`
	tagS        = `[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}`
	digestS     = `[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}`
	refRE       = regexp.MustCompile(`^(?:(` + registryS + `)` + regexp.QuoteMeta(`/`) + `)?` +
		`(` + repoPartS + `(?:` + regexp.QuoteMeta(`/`) + repoPartS + `)*)` +
		`(?:` + regexp.QuoteMeta(`:`) + `(` + tagS + `))?` +
		`(?:` + regexp.QuoteMeta(`@`) + `(` + digestS + `))?$`)
)

// imageSplit parses an imageRef into a registry, repository, tag, digest, or an error.
func imageSplit(imageRef string) (string, string, string, string, error) {
	imageSplit := refRE.FindStringSubmatch(imageRef)
	if len(imageSplit) < 5 {
		return "", "", "", "", fmt.Errorf("failed to parse ref %s", imageRef)
	}
	return imageSplit[1], imageSplit[2], imageSplit[3], imageSplit[4], nil
}

// DeleteImage deletes an image reference from the Registry container.
// It will use the HTTP endpoint of the Registry container to delete it,
// doing a HEAD request to get the image digest and then a DELETE request
// to actually delete the image.
// E.g. imageRef = "localhost:5000/alpine:latest"
func (c *RegistryContainer) DeleteImage(ctx context.Context, imageRef string) error {
	_, repo, tag, digest, err := imageSplit(imageRef)
	if err != nil {
		return err
	}
	if tag == "" && digest == "" {
		tag = "latest"
	}
	deleteEndpoint := fmt.Sprintf("/v2/%s/manifests/%s", repo, tag)
	if digest != "" {
		deleteEndpoint = fmt.Sprintf("/v2/%s/manifests/%s", repo, digest)
	}
	return wait.ForHTTP(deleteEndpoint).
		WithMethod(http.MethodDelete).
		WithStatusCodeMatcher(func(statusCode int) bool {
			return statusCode == http.StatusAccepted
		}).
		WaitUntilReady(ctx, c)
}

// ImageExists checks if an image exists in the Registry container. It will use the v2 HTTP endpoint
// of the Registry container to check if the image reference exists.
// E.g. imageRef = "localhost:5000/alpine:latest"
func (c *RegistryContainer) ImageExists(ctx context.Context, imageRef string) error {
	_, repo, tag, digest, err := imageSplit(imageRef)
	if err != nil {
		return err
	}
	if tag == "" && digest == "" {
		tag = "latest"
	}
	endpoint := fmt.Sprintf("/v2/%s/manifests/%s", repo, tag)
	if digest != "" {
		endpoint = fmt.Sprintf("/v2/%s/manifests/%s", repo, digest)
	}

	return wait.ForHTTP(endpoint).
		WithMethod(http.MethodHead).
		WithForcedIPv4LocalHost().
		WithStatusCodeMatcher(func(statusCode int) bool {
			return statusCode == http.StatusOK
		}).
		WithResponseHeadersMatcher(func(headers http.Header) bool {
			return headers.Get("Docker-Content-Digest") != ""
		}).
		WaitUntilReady(ctx, c)
}

// PushImage pushes an image to the Registry container. It will use the internally stored RegistryName
// to push the image to the container, and it will finally wait for the image to be pushed.
func (c *RegistryContainer) PushImage(ctx context.Context, ref string) error {
	// TODO(bmitch): either add support with regctl, drop function (depending on initialization directory), or reuse docker CLI
	// dockerProvider, err := testcontainers.NewDockerProvider()
	// if err != nil {
	// 	return fmt.Errorf("failed to create Docker provider: %w", err)
	// }
	// defer dockerProvider.Close()

	// dockerCli := dockerProvider.Client()

	// _, imageAuth, err := testcontainers.DockerImageAuth(ctx, ref)
	// if err != nil {
	// 	return fmt.Errorf("failed to get image auth: %w", err)
	// }

	// pushOpts := types.ImagePushOptions{
	// 	All: true,
	// }

	// // see https://github.com/docker/docs/blob/e8e1204f914767128814dca0ea008644709c117f/engine/api/sdk/examples.md?plain=1#L649-L657
	// encodedJSON, err := json.Marshal(imageAuth)
	// if err != nil {
	// 	return fmt.Errorf("failed to encode image auth: %w", err)
	// } else {
	// 	pushOpts.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
	// }

	// _, err = dockerCli.ImagePush(ctx, ref, pushOpts)
	// if err != nil {
	// 	return fmt.Errorf("failed to push image %s: %w", ref, err)
	// }

	// return c.ImageExists(ctx, ref)
	return fmt.Errorf("not implemented")
}

// RunContainer creates an instance of the Registry container type
func RunContainer(ctx context.Context, opts ...testcontainers.ContainerCustomizer) (*RegistryContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        olaregImage,
		ExposedPorts: []string{olaregPort},
		Cmd: []string{
			"serve",
			"--store-type", "mem",
			"--api-delete",
			"--dir", containerRegistryPath,
		},
		WaitingFor: wait.ForAll(
			wait.ForExposedPort(),
			wait.ForHTTP("/v2/"),
		),
	}

	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&genericContainerReq); err != nil {
			return nil, err
		}
	}

	container, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	if err != nil {
		return nil, err
	}

	c := &RegistryContainer{Container: container}

	address, err := c.Address(ctx)
	if err != nil {
		return c, err
	}

	c.RegistryName = strings.TrimPrefix(address, "http://")

	return c, nil
}
