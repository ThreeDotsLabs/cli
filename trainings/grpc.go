package trainings

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func newRequestHeader(cliMetadata CliMetadata) grpc.CallOption {
	md := metadata.New(map[string]string{
		"cli-version":  cliMetadata.Version,
		"cli-commit":   cliMetadata.Commit,
		"os":           cliMetadata.OS,
		"architecture": cliMetadata.Architecture,
		"command":      cliMetadata.ExecutedCommand,
	})

	return grpc.Header(&md)
}
