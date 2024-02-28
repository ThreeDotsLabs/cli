package trainings

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) ConfigureGlobally(ctx context.Context, token, serverAddr, region string, insecure bool) error {
	if region != "" {
		if region != "eu" && region != "us" {
			return errors.New("region can be only eu or us")
		}
	}

	resp, err := h.newGrpcClientWithAddr(ctx, serverAddr, region, insecure).Init(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.InitRequest{
			Token: token,
		},
	)
	if err != nil {
		return err
	}

	if region == "" {
		region = resp.Region
	}

	return h.config.WriteGlobalConfig(config.GlobalConfig{
		Token:      token,
		ServerAddr: serverAddr,
		Region:     region,
		Insecure:   insecure,
	})
}
