package trainings

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ThreeDotsLabs/cli/tdl/trainings/config"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/files"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

type Handlers struct {
	config config.Config
	files  files.Files

	grpcClient genproto.ServerClient
}

func NewHandlers() *Handlers {
	conf := config.NewConfig()

	return &Handlers{
		config: conf,
		files:  files.NewDefaultFiles(),
	}
}

func (h *Handlers) newGrpcClient(ctx context.Context) genproto.ServerClient {
	return h.newGrpcClientWithAddr(ctx, h.config.GlobalConfig().ServerAddr)
}

func (h *Handlers) newGrpcClientWithAddr(ctx context.Context, addr string) genproto.ServerClient {
	if h.grpcClient == nil {
		conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}

		h.grpcClient = genproto.NewServerClient(conn)
	}

	return h.grpcClient
}
