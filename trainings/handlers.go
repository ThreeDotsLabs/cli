package trainings

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
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
	globalConfig := h.config.GlobalConfig()

	return h.newGrpcClientWithAddr(ctx, globalConfig.ServerAddr, globalConfig.Insecure)
}

func (h *Handlers) newGrpcClientWithAddr(ctx context.Context, addr string, insecure bool) genproto.ServerClient {
	if h.grpcClient == nil {
		var opts []grpc.DialOption

		if insecure {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
		} else {
			systemRoots, err := x509.SystemCertPool()
			if err != nil {
				panic(errors.Wrap(err, "cannot load root CA cert"))
			}
			creds := credentials.NewTLS(&tls.Config{
				RootCAs:    systemRoots,
				MinVersion: tls.VersionTLS12,
			})
			opts = append(opts, grpc.WithTransportCredentials(creds))
		}

		conn, err := grpc.DialContext(ctx, addr, opts...)

		if err != nil {
			panic(err)
		}

		h.grpcClient = genproto.NewServerClient(conn)
	}

	return h.grpcClient
}
