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

	resp, err := h.newGrpcClientWithAddr(serverAddr, region, insecure).Init(
		ctx,
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

	// Read existing config to preserve MCP settings (and any future fields).
	// ConfiguredGlobally() check: on first-ever configure, there's no file yet.
	var globalCfg config.GlobalConfig
	if h.config.ConfiguredGlobally() {
		globalCfg = h.config.GlobalConfig()
	}
	globalCfg.Token = token
	globalCfg.ServerAddr = serverAddr
	globalCfg.Region = region
	globalCfg.Insecure = insecure
	return h.config.WriteGlobalConfig(globalCfg)
}
