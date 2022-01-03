package trainings

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

const defaultTrainingsServer = "academy-grpc.threedots.tech:443"

func (h *Handlers) ConfigureGlobally(ctx context.Context, token, serverAddr string, override, insecure bool) error {
	if !override && h.config.ConfiguredGlobally() {
		return errors.New("trainings are already configured. Please pass --override flag to configure again")
	}

	if serverAddr == "" {
		serverAddr = defaultTrainingsServer
	}

	if _, err := h.newGrpcClientWithAddr(ctx, serverAddr, insecure).Init(
		context.Background(),
		&genproto.InitRequest{Token: token},
	); err != nil {
		return err
	}

	return h.config.WriteGlobalConfig(config.GlobalConfig{
		Token:      token,
		ServerAddr: serverAddr,
		Insecure:   insecure,
	})
}
