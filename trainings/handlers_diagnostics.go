package trainings

import (
	"context"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/diagnostics"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

// Diagnostics runs the connectivity diagnostics suite and prints a report.
// It is wired by the `tdl training diagnostics` subcommand and must run even
// when the user is not configured — bypassing the parent training command's
// implicit connection check.
func (h *Handlers) Diagnostics(ctx context.Context, server, region string, insecure bool) error {
	return diagnostics.Run(ctx, diagnostics.Options{
		Server:             server,
		Region:             region,
		Insecure:           insecure,
		ConfiguredGlobally: h.config.ConfiguredGlobally(),
		GlobalConfig: func() config.GlobalConfig {
			return h.config.GlobalConfig()
		},
		CLIVersion: h.cliMetadata.Version,
		CLICommit:  h.cliMetadata.Commit,
		OS:         h.cliMetadata.OS,
		Arch:       h.cliMetadata.Architecture,
		GoVersion:  h.cliMetadata.GoVersion,
		BuildFreshGRPCClient: func(addr, region string, insecure bool) (genproto.TrainingsClient, func() error, error) {
			return h.newFreshGrpcClient(addr, region, insecure)
		},
		ResolveAddr:    resolveServerAddr,
		BuildTLSConfig: buildTLSConfig,
	})
}
