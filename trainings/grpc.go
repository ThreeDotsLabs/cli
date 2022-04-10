package trainings

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func ctxWithRequestHeader(ctx context.Context, cliMetadata CliMetadata) context.Context {
	md := metadata.New(map[string]string{
		"cli-version":  cliMetadata.Version,
		"cli-commit":   cliMetadata.Commit,
		"os":           cliMetadata.OS,
		"architecture": cliMetadata.Architecture,
		"command":      cliMetadata.ExecutedCommand,
	})

	return metadata.NewOutgoingContext(ctx, md)
}
