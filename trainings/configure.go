package trainings

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) ConfigureGlobally(ctx context.Context, token, serverAddr, region string, override, insecure bool) error {
	if !override && h.config.ConfiguredGlobally() {
		return errors.New("trainings are already configured. Please pass --override flag to configure again")
	}

	if region != "" {
		if region != "eu" && region != "us" {
			return errors.New("region can be only eu or us")
		}
	}

	if _, err := h.newGrpcClientWithAddr(ctx, serverAddr, region, insecure).Init(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.InitRequest{
			Token: token,
		},
	); err != nil {
		return err
	}

	return h.config.WriteGlobalConfig(config.GlobalConfig{
		Token:      token,
		ServerAddr: serverAddr,
		Region:     region,
		Insecure:   insecure,
	})
}
