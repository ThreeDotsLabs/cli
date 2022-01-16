package trainings

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

type Handlers struct {
	config config.Config

	grpcClient genproto.ServerClient
}

func NewHandlers() *Handlers {
	conf := config.NewConfig()

	return &Handlers{
		config: conf,
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

func newTrainingRootFs(trainingRoot string) afero.Fs {
	// Privacy of your files is our priority.
	//
	// We should never trust the remote server.
	// Writing files based on external name is a vector for Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	//
	// To avoid that we are using afero.BasePathFs with base on training root for all operations in trainings dir.
	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), trainingRoot)
	return trainingRootFs
}
