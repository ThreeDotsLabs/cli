// Package diagnostics runs ordered connectivity checks against the verification
// server and prints an interpreted report. It is the implementation of the
// `tdl training diagnostics` subcommand.
//
// Checks run in fixed order: snapshot, proxy env, DNS, TCP, TLS, HTTPS, gRPC
// Ping, latency probe, clock skew. Each later layer is skipped only when the
// snapshot can't even be assembled (no network info), otherwise every check
// runs and contributes to the interpretation.
package diagnostics

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

// Options carries everything Run needs from the parent handler. Closures are
// used so the diagnostics package does not import trainings/.
type Options struct {
	Server, Region string
	Insecure       bool

	ConfiguredGlobally bool
	GlobalConfig       func() config.GlobalConfig

	CLIVersion, CLICommit string
	OS, Arch, GoVersion   string

	BuildFreshGRPCClient func(addr, region string, insecure bool) (genproto.TrainingsClient, func() error, error)
	ResolveAddr          func(addr, region string) string
	BuildTLSConfig       func(insecure bool) *tls.Config
}

// token returns the token from GlobalConfig if the user is configured. Never
// included in any rendered output — only forwarded to RPC requests that need it.
func (o Options) token() string {
	if !o.ConfiguredGlobally || o.GlobalConfig == nil {
		return ""
	}
	return safeGlobalConfig(o.GlobalConfig).Token
}

// Run executes all diagnostic checks and prints the report. It always returns
// nil — diagnostic findings are reported in the output, not as errors.
func Run(ctx context.Context, opts Options) error {
	r := newRenderer()

	snap := buildSnapshot(opts)
	r.printHeader()
	r.printSnapshot(snap)

	const total = 10
	results := make([]Result, 0, total)

	step := func(idx int, name string, fn func(context.Context) Result) Result {
		r.startStep(idx, total, name)
		start := time.Now()
		res := fn(ctx)
		res.Name = name
		if res.Duration == 0 {
			res.Duration = time.Since(start)
		}
		r.finishStep(idx, total, res)
		results = append(results, res)
		return res
	}

	step(1, NameProxy, checkProxyEnv)
	step(2, NameDNS, func(ctx context.Context) Result { return checkDNS(ctx, snap.Host) })
	step(3, NameTCP, func(ctx context.Context) Result { return checkTCP(ctx, snap.Host, snap.Port) })
	step(4, NameTLS, func(ctx context.Context) Result {
		return checkTLS(ctx, snap.Host, snap.Port, opts.BuildTLSConfig(opts.Insecure), opts.Insecure)
	})
	httpsRes := step(5, NameHTTPS, func(ctx context.Context) Result {
		return checkHTTPS(ctx, internal.WebsiteAddress)
	})
	step(6, NamePing, func(ctx context.Context) Result {
		return checkGRPCPing(ctx, opts.BuildFreshGRPCClient, opts.Server, opts.Region, opts.Insecure)
	})
	step(7, NameGetTrainings, func(ctx context.Context) Result {
		return checkGetTrainings(ctx, opts.BuildFreshGRPCClient, opts.Server, opts.Region, opts.Insecure, snap.Configured)
	})
	step(8, NameStream, func(ctx context.Context) Result {
		return checkStreaming(ctx, opts.BuildFreshGRPCClient, opts.Server, opts.Region, opts.Insecure, snap.Configured, opts.token())
	})
	step(9, NameLat, func(ctx context.Context) Result {
		return checkLatency(ctx, opts.BuildFreshGRPCClient, opts.Server, opts.Region, opts.Insecure)
	})
	step(10, NameClock, func(ctx context.Context) Result {
		return checkClockSkew(httpsRes)
	})

	r.printSummary(results)
	r.printInterpretation(Interpret(results, snap))
	return nil
}

// Snapshot is the captured environment used by the renderer header and the
// interpreter (e.g. to surface "HTTPS_PROXY is set" alongside a TLS failure).
type Snapshot struct {
	CLIVersion, CLICommit string
	OS, Arch, GoVersion   string

	Configured  bool
	Server      string
	Region      string
	Insecure    bool
	TokenStatus string

	// Effective dial target with port (e.g. "eu.academy-grpc.threedots.tech:443").
	Target string
	// Host without port and the port (parsed from Target).
	Host string
	Port string

	ProxyEnv map[string]string

	// Tunnels lists active interface names that look like VPNs / tunnels
	// (utun, tun, ppp, tailscale, wg, gpd). Diagnostic-only — surfaced in the
	// snapshot and referenced by the interpreter when network checks fail.
	Tunnels []string
}

func buildSnapshot(opts Options) Snapshot {
	s := Snapshot{
		CLIVersion: opts.CLIVersion,
		CLICommit:  opts.CLICommit,
		OS:         opts.OS,
		Arch:       opts.Arch,
		GoVersion:  opts.GoVersion,
		Configured: opts.ConfiguredGlobally,
		Server:     opts.Server,
		Region:     opts.Region,
		Insecure:   opts.Insecure,
	}

	// If user did not pass --server / --region, fall back to global config when configured.
	if opts.ConfiguredGlobally && opts.GlobalConfig != nil {
		// GlobalConfig() panics on missing token; diagnostics must not crash on a malformed config.
		gc := safeGlobalConfig(opts.GlobalConfig)
		if s.Server == "" {
			s.Server = gc.ServerAddr
		}
		if s.Region == "" {
			s.Region = gc.Region
		}
		if !s.Insecure {
			s.Insecure = gc.Insecure
		}
		s.TokenStatus = tokenStatus(gc.Token)
	} else {
		s.TokenStatus = "missing (not configured)"
	}

	s.Target = opts.ResolveAddr(s.Server, s.Region)
	s.Host, s.Port = splitHostPort(s.Target)
	s.ProxyEnv = readProxyEnv()
	s.Tunnels = detectTunnels()
	return s
}

func tokenStatus(tok string) string {
	if tok == "" {
		return "missing"
	}
	trimmed := strings.TrimSpace(tok)
	if trimmed != tok {
		return fmt.Sprintf("present (length=%d, has leading/trailing whitespace)", len(tok))
	}
	return fmt.Sprintf("present (length=%d)", len(tok))
}

// safeGlobalConfig wraps GlobalConfig() so its panic-on-malformed behavior does
// not crash diagnostics — the whole point is to run on broken setups.
func safeGlobalConfig(get func() config.GlobalConfig) (cfg config.GlobalConfig) {
	defer func() { _ = recover() }()
	cfg = get()
	return
}

