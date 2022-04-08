package trainings

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) ConfigureGlobally(ctx context.Context, token, serverAddr string, override, insecure bool) error {
	if !override && h.config.ConfiguredGlobally() {
		return errors.New("trainings are already configured. Please pass --override flag to configure again")
	}

	if _, err := h.newGrpcClientWithAddr(ctx, serverAddr, insecure).Init(
		context.Background(),
		&genproto.InitRequest{
			Header: newRequestHeader(h.cliMetadata),
			Token:  token,
		},
	); err != nil {
		return err
	}

	return h.config.WriteGlobalConfig(config.GlobalConfig{
		Token:      token,
		ServerAddr: serverAddr,
		Insecure:   insecure,
	})
}
