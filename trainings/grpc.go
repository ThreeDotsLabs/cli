package trainings

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// debugHeaders builds the metadata sent with every gRPC request.
// Called per-RPC so training config changes mid-session are reflected.
func (h *Handlers) debugHeaders() metadata.MD {
	md := metadata.New(map[string]string{
		"cli-version":  h.cliMetadata.Version,
		"cli-commit":   h.cliMetadata.Commit,
		"os":           h.cliMetadata.OS,
		"os-version":   h.cliMetadata.OSVersion,
		"architecture": h.cliMetadata.Architecture,
		"go-version":   h.cliMetadata.GoVersion,
		"git-version":  h.cliMetadata.GitVersion,
		"command":      h.cliMetadata.ExecutedCommand,
		"interactive":  fmt.Sprint(h.cliMetadata.Interactive),
	})

	h.appendTrainingHeaders(md)

	return md
}

// appendTrainingHeaders adds git integration settings when running inside a training directory.
// Uses recover() because TrainingConfig() panics on corrupt TOML.
func (h *Handlers) appendTrainingHeaders(md metadata.MD) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("panic", r).Debug("Could not read training config for debug headers")
		}
	}()

	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		return
	}

	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), trainingRoot).(*afero.BasePathFs)
	cfg := h.config.TrainingConfig(trainingRootFs)

	md.Set("git-enabled", fmt.Sprint(cfg.GitConfigured && cfg.GitEnabled))
	md.Set("git-auto-commit", fmt.Sprint(cfg.GitAutoCommit))
	md.Set("git-auto-golden", fmt.Sprint(cfg.GitAutoGolden))
	md.Set("git-golden-mode", cfg.GitGoldenMode)
}

func (h *Handlers) unaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.NewOutgoingContext(ctx, h.debugHeaders())
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (h *Handlers) streamInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = metadata.NewOutgoingContext(ctx, h.debugHeaders())
		return streamer(ctx, desc, cc, method, opts...)
	}
}
