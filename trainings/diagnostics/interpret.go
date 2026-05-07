package diagnostics

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/internal"
)

// Check names — kept as constants so the interpreter, renderer, and orchestrator agree.
const (
	NameProxy        = "Proxy environment"
	NameDNS          = "DNS resolution"
	NameTCP          = "TCP connect"
	NameTLS          = "TLS handshake"
	NameHTTPS        = "HTTPS GET (academy website)"
	NamePing         = "gRPC Ping"
	NameGetTrainings = "gRPC GetTrainings (auth)"
	NameStream       = "gRPC VerifyExercise stream"
	NameLat          = "Latency probe (5 pings)"
	NameClock        = "Clock skew"
)

// Interpret reads the full result set and the snapshot and returns a single
// most-actionable user-facing message followed by supporting notes.
//
// Order of priority: layer-by-layer from outermost broken layer down. The first
// failing layer is the headline; the rest are supporting context.
func Interpret(results []Result, snap Snapshot) string {
	idx := indexResults(results)

	var b strings.Builder

	switch {
	case failed(idx, NameDNS):
		r := idx[NameDNS]
		if r.Extras["system_dns_broken"] == "1" {
			writeHeadline(&b, "Your system DNS is broken — public DNS works fine.")
			writeBody(&b,
				fmt.Sprintf("System resolver could not look up %q, but querying 8.8.8.8 directly succeeds.", snap.Host),
				"This points at your local DNS configuration:",
				"  - Corporate / VPN DNS server is broken or doesn't have this domain.",
				"  - Pi-hole / Adguard / similar resolver is blocking the lookup.",
				"  - /etc/resolv.conf points at an unreachable nameserver.",
			)
		} else {
			writeHeadline(&b, "DNS resolution failed.")
			writeBody(&b,
				fmt.Sprintf("The hostname %q could not be resolved to an IP.", snap.Host),
				"Likely causes:",
				"  - No internet connection, or you're behind a captive portal that hasn't been accepted.",
				"  - Custom DNS (corporate, VPN, Pi-hole) is blocking or failing the lookup.",
				"  - A stale entry in /etc/hosts is overriding the real address.",
				fmt.Sprintf("Try: %s", color.CyanString("nslookup "+snap.Host)),
			)
		}

	case failed(idx, NameTCP):
		writeHeadline(&b, fmt.Sprintf("Cannot reach %s on port %s.", snap.Host, snap.Port))
		writeBody(&b,
			"DNS resolves but a TCP connection cannot be established. This usually means:",
			"  - A firewall, VPN, or corporate proxy is blocking outbound traffic to this port.",
			"  - The route to the server is broken (try a different network / disable VPN).",
		)
		if hasProxy(snap) {
			writeBody(&b, color.YellowString("You have proxy environment variables set; they may be the cause."))
		}

	case failed(idx, NameTLS):
		r := idx[NameTLS]
		switch {
		case r.Extras["alpn_not_h2"] == "1":
			alpn := r.Extras["alpn"]
			if alpn == "" {
				alpn = "(none)"
			}
			writeHeadline(&b, "TLS handshake works, but ALPN did not negotiate h2 (HTTP/2).")
			writeBody(&b,
				fmt.Sprintf("The server (or a middlebox between you and it) negotiated %q instead. gRPC requires HTTP/2.", alpn),
				"This typically means a corporate proxy is intercepting traffic and downgrading the protocol.",
				"Subsequent gRPC checks will fail or hang as a consequence.",
			)
		case r.Extras["tls_unknown_authority"] == "1":
			writeHeadline(&b, "TLS handshake rejected the server certificate (unknown authority).")
			writeBody(&b,
				"This is typically a corporate TLS-interception proxy (Zscaler, Netskope, Bluecoat, etc.).",
				"Ask your IT team for the proxy's root CA and install it in your system trust store.",
				fmt.Sprintf("To confirm the diagnosis, you can re-run with %s — but never use that flag for real traffic.", color.CyanString("--insecure")),
			)
		case r.Extras["tls_cert_time"] == "1":
			writeHeadline(&b, "TLS handshake failed: certificate time-validity issue.")
			writeBody(&b,
				"Either the server certificate has expired, or your system clock is wrong.",
				fmt.Sprintf("Check your clock with %s and verify NTP is running.", color.CyanString("date")),
			)
		case r.Extras["tls_hostname_mismatch"] == "1":
			writeHeadline(&b, "TLS handshake failed: hostname mismatch.")
			writeBody(&b,
				"The certificate the server presented does not cover the hostname you connected to.",
				"This can happen with DNS misdirection, transparent proxies, or a typo in --server.",
			)
		default:
			writeHeadline(&b, "TLS handshake failed.")
			writeBody(&b, fmt.Sprintf("Error: %v", r.Err))
		}

	case failed(idx, NamePing):
		r := idx[NamePing]
		switch status.Code(r.Err) {
		case codes.Unauthenticated:
			writeHeadline(&b, "The server rejected your token.")
			writeBody(&b,
				fmt.Sprintf("Run %s with a fresh token from %s.",
					color.CyanString(internal.BinaryName()+" training configure <token>"),
					internal.WebsiteAddress,
				),
			)
		case codes.DeadlineExceeded:
			writeHeadline(&b, "gRPC Ping timed out.")
			writeBody(&b, "The lower layers worked, but the request did not complete in time.",
				"This matches the symptom of an unstable connection (see latency probe below).")
		case codes.Unavailable:
			writeHeadline(&b, "gRPC reports the server as Unavailable.")
			writeBody(&b, "The transport is up but the gRPC connection cannot be established. The server may be temporarily down, or a proxy is interfering.")
		default:
			writeHeadline(&b, "gRPC Ping failed.")
			writeBody(&b, fmt.Sprintf("gRPC code: %s — %v", status.Code(r.Err), r.Err))
		}

	case failed(idx, NameGetTrainings):
		r := idx[NameGetTrainings]
		switch status.Code(r.Err) {
		case codes.Unauthenticated:
			writeHeadline(&b, "The server rejected your token.")
			writeBody(&b,
				"Transport (Ping) succeeded, but the authenticated request was rejected.",
				fmt.Sprintf("Run %s with a fresh token from %s.",
					color.CyanString(internal.BinaryName()+" training configure <token>"),
					internal.WebsiteAddress,
				),
			)
		case codes.DeadlineExceeded:
			writeHeadline(&b, "Authenticated gRPC request timed out.")
			writeBody(&b, "Ping succeeded but a real call did not. The connection may be unstable on larger requests, or the server is slow.")
		default:
			writeHeadline(&b, "Authenticated gRPC request failed.")
			writeBody(&b, fmt.Sprintf("gRPC code: %s — %v", status.Code(r.Err), r.Err))
		}

	case failed(idx, NameStream):
		r := idx[NameStream]
		if r.Extras["stream_deadline"] == "1" {
			writeHeadline(&b, "Streaming gRPC request timed out.")
			writeBody(&b,
				"Unary gRPC calls work, but a streaming call hangs. This is the most common cause of intermittent DeadlineExceeded errors during exercise verification.",
				"Likely culprits:",
				"  - HTTP/2 flow control mishandled by a proxy.",
				"  - Idle-timeout on a corporate proxy that closes streams it considers stale.",
				"  - Unstable network drops the long-lived connection.",
			)
		} else {
			writeHeadline(&b, "Streaming gRPC request failed.")
			writeBody(&b, fmt.Sprintf("Error: %v", r.Err))
		}

	case failed(idx, NameLat):
		r := idx[NameLat]
		loss := r.Extras["loss"]
		writeHeadline(&b, fmt.Sprintf("Connection is unstable: %s of 5 pings failed.", loss))
		writeBody(&b,
			"This matches the packet-loss symptom that causes intermittent DeadlineExceeded errors.",
			"Try: switch to a wired connection, disable VPN, or move closer to the access point.",
		)

	case failed(idx, NameClock):
		writeHeadline(&b, "Your system clock is significantly off.")
		writeBody(&b,
			"This can break TLS handshakes and authentication.",
			"Verify NTP is enabled and resync your clock.",
		)

	case len(idx) > 0:
		writeHeadline(&b, color.GreenString("All checks passed."))
		writeBody(&b,
			"If you are still seeing DeadlineExceeded errors, the issue may be intermittent.",
			fmt.Sprintf("Re-run with %s for verbose logs and share this output if you contact support.",
				color.CyanString("--verbose")),
		)
	}

	if hasProxy(snap) {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, color.YellowString("Note: proxy environment variables are set. "+
			"They may be involved if anything above failed — "+
			"these settings affect how Go's HTTP and gRPC clients reach the server."))
	}

	if len(snap.Tunnels) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, color.YellowString(
			"Note: VPN / tunnel interface(s) active: %s. "+
				"VPNs are a common cause of TLS interception, broken IPv6, and stalled streaming RPCs — "+
				"if anything above failed, try disconnecting and re-running.",
			strings.Join(snap.Tunnels, ", "),
		))
	}

	return strings.TrimRight(b.String(), "\n")
}

func indexResults(results []Result) map[string]Result {
	m := make(map[string]Result, len(results))
	for _, r := range results {
		m[r.Name] = r
	}
	return m
}

func failed(idx map[string]Result, name string) bool {
	r, ok := idx[name]
	return ok && !r.Pass && !r.Skipped
}

func hasProxy(snap Snapshot) bool {
	return len(snap.ProxyEnv) > 0
}

func writeHeadline(b *strings.Builder, s string) {
	fmt.Fprintln(b, color.New(color.Bold).Sprint(s))
}

func writeBody(b *strings.Builder, lines ...string) {
	for _, l := range lines {
		fmt.Fprintln(b, l)
	}
}
