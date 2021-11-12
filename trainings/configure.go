package trainings

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/cli/tdl/trainings/config"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

func (h *Handlers) ConfigureGlobally(ctx context.Context, token, serverAddr string, override bool) error {
	if !override && h.config.ConfiguredGlobally() {
		return errors.New("trainings are already configured. Please pass --override flag to configure again")
	}

	if _, err := h.newGrpcClientWithAddr(ctx, serverAddr).Init(
		context.Background(),
		&genproto.InitRequest{Token: token},
	); err != nil {
		return err
	}

	return h.config.WriteGlobalConfig(config.GlobalConfig{
		Token:      token,
		ServerAddr: serverAddr,
	})
}
