package trainings

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

type Handlers struct {
	config config.Config

	grpcClient  genproto.TrainingsClient
	cliMetadata CliMetadata

	solutionHintDisplayed bool
	solutionAvailable     bool
	stuckRunCount         int
	notifications         map[string]struct{}
}

type CliMetadata struct {
	Version string
	Commit  string

	Architecture string
	OS           string
	OSVersion    string
	GoVersion    string
	GitVersion   string

	ExecutedCommand string
	Interactive     bool
}

func NewHandlers(cliVersion CliMetadata) *Handlers {
	conf := config.NewConfig()

	return &Handlers{
		config:        conf,
		cliMetadata:   cliVersion,
		notifications: map[string]struct{}{},
	}
}

func (h *Handlers) newGrpcClient() genproto.TrainingsClient {
	globalConfig := h.config.GlobalConfig()

	return h.newGrpcClientWithAddr(globalConfig.ServerAddr, globalConfig.Region, globalConfig.Insecure)
}

func (h *Handlers) newGrpcClientWithAddr(addr string, region string, insecure bool) genproto.TrainingsClient {
	if addr == "" {
		addr = internal.DefaultTrainingsServer
	}

	if region != "" {
		addr = fmt.Sprintf("%s.%s", region, addr)
	}

	if h.grpcClient == nil {
		var opts []grpc.DialOption

		if insecure {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
		} else {
			systemRoots, err := x509.SystemCertPool()
			if err != nil && runtime.GOOS != "windows" {
				panic(errors.Wrap(err, "cannot load root CA cert"))
			}
			if systemRoots == nil {
				systemRoots = x509.NewCertPool()
			}
			creds := credentials.NewTLS(&tls.Config{
				RootCAs:    systemRoots,
				MinVersion: tls.VersionTLS12,
			})
			opts = append(opts, grpc.WithTransportCredentials(creds))
		}

		opts = append(opts,
			grpc.WithUnaryInterceptor(h.unaryInterceptor()),
			grpc.WithStreamInterceptor(h.streamInterceptor()),
		)

		conn, err := grpc.NewClient(addr, opts...)

		if err != nil {
			panic(err)
		}

		h.grpcClient = genproto.NewTrainingsClient(conn)
	}

	return h.grpcClient
}

func (h *Handlers) newGitOps() *git.Ops {
	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		logrus.WithError(err).Debug("Can't find training root for git ops")
		return git.NewOps("", true) // disabled
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	cfg := h.config.TrainingConfig(trainingRootFs)
	disabled := !cfg.GitConfigured || !cfg.GitEnabled
	return git.NewOps(trainingRoot, disabled)
}

func newTrainingRootFs(trainingRoot string) *afero.BasePathFs {
	// Privacy of your files is our priority.
	//
	// We should never trust the remote server.
	// Writing files based on external name is a vector for Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	//
	// To avoid that we are using afero.BasePathFs with base on training root for all operations in trainings dir.
	return afero.NewBasePathFs(afero.NewOsFs(), trainingRoot).(*afero.BasePathFs)
}

// CheckServerConnection verifies we can reach the server before running a command.
// Explicit serverAddr/region/insecure params take priority over global config, which takes priority over defaults.
func (h *Handlers) CheckServerConnection(ctx context.Context, serverAddr, region string, insecure bool) error {
	var client genproto.TrainingsClient
	if serverAddr != "" {
		client = h.newGrpcClientWithAddr(serverAddr, region, insecure)
	} else if h.config.ConfiguredGlobally() {
		client = h.newGrpcClient()
	} else {
		client = h.newGrpcClientWithAddr(internal.DefaultTrainingsServer, "", false)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := client.Ping(pingCtx, &emptypb.Empty{})
	if err != nil {
		return formatConnectionError(err)
	}
	return nil
}
