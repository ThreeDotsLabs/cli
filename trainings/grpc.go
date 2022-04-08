package trainings

import "github.com/ThreeDotsLabs/cli/trainings/genproto"

func newRequestHeader(cliMetadata CliMetadata) *genproto.RequestHeader {
	return &genproto.RequestHeader{
		CliVersion:   cliMetadata.Version,
		CliCommit:    cliMetadata.Commit,
		Os:           cliMetadata.OS,
		Architecture: cliMetadata.Architecture,
		Command:      cliMetadata.ExecutedCommand,
	}
}
