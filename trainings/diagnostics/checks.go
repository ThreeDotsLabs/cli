package diagnostics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

// Result is the outcome of a single diagnostic check.
type Result struct {
	Name     string
	Pass     bool
	Skipped  bool
	Detail   string
	Err      error
	Duration time.Duration
	Subitems []string
	// Extras are extra fields the interpreter inspects (e.g. TLS issuer string).
	Extras map[string]string
}

func ok(detail string, sub ...string) Result {
	return Result{Pass: true, Detail: detail, Subitems: sub}
}
func fail(err error, detail string, sub ...string) Result {
	return Result{Pass: false, Err: err, Detail: detail, Subitems: sub}
}

// proxyEnvVars are the env vars users typically set for HTTP/HTTPS proxies.
var proxyEnvVars = []string{
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY",
	"http_proxy", "https_proxy", "no_proxy", "all_proxy",
}

func readProxyEnv() map[string]string {
	out := map[string]string{}
	for _, k := range proxyEnvVars {
		if v := os.Getenv(k); v != "" {
			out[k] = v
		}
	}
	return out
}

func checkProxyEnv(_ context.Context) Result {
	envs := readProxyEnv()
	if len(envs) == 0 {
		return ok("no proxy environment variables set")
	}
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sub := make([]string, 0, len(keys))
	for _, k := range keys {
		sub = append(sub, fmt.Sprintf("%s=%s", k, envs[k]))
	}
	return Result{
		Pass:     true,
		Detail:   fmt.Sprintf("%d proxy variable(s) set — may affect connectivity", len(keys)),
		Subitems: sub,
		Extras:   map[string]string{"proxy_set": "1"},
	}
}

func splitHostPort(target string) (host, port string) {
	h, p, err := net.SplitHostPort(target)
	if err != nil {
		return target, "443"
	}
	return h, p
}

func checkDNS(ctx context.Context, host string) Result {
	if host == "" {
		return fail(nil, "no host to resolve")
	}
	dnsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sysIPs, sysErr := net.DefaultResolver.LookupIPAddr(dnsCtx, host)
	pubIPs, pubErr := publicResolver().LookupIPAddr(dnsCtx, host)

	sub := make([]string, 0, 4)
	if sysErr != nil {
		sub = append(sub, fmt.Sprintf("system: failed (%v)", sysErr))
	} else {
		sub = append(sub, fmt.Sprintf("system: %s", joinIPs(sysIPs)))
	}
	if pubErr != nil {
		sub = append(sub, fmt.Sprintf("public (8.8.8.8): failed (%v)", pubErr))
	} else {
		sub = append(sub, fmt.Sprintf("public (8.8.8.8): %s", joinIPs(pubIPs)))
	}

	extras := map[string]string{}
	switch {
	case sysErr != nil && pubErr == nil:
		// Strongest signal: system DNS broken, public DNS works.
		extras["system_dns_broken"] = "1"
		return Result{
			Pass:     false,
			Err:      sysErr,
			Detail:   "system DNS failed but public DNS resolved — your DNS configuration is the problem",
			Subitems: sub,
			Extras:   extras,
		}
	case sysErr != nil && pubErr != nil:
		return Result{
			Pass:     false,
			Err:      sysErr,
			Detail:   fmt.Sprintf("DNS lookup failed: %v", sysErr),
			Subitems: sub,
		}
	case sysErr == nil && pubErr != nil:
		// Public DNS blocked is common in corporate networks; not a failure
		// since the system resolver works — but mention it.
		return Result{
			Pass:     true,
			Detail:   fmt.Sprintf("resolved %d IP(s) (public DNS blocked, but not needed)", len(sysIPs)),
			Subitems: sub,
		}
	}

	// Both resolvers worked. Flag if they returned wildly different results
	// (could indicate /etc/hosts pollution or DNS hijacking).
	if !ipSetsOverlap(sysIPs, pubIPs) {
		extras["dns_mismatch"] = "1"
		return Result{
			Pass:     true,
			Detail:   "system and public DNS returned different IPs — possible /etc/hosts override or DNS hijack",
			Subitems: sub,
			Extras:   extras,
		}
	}
	return Result{
		Pass:     true,
		Detail:   fmt.Sprintf("resolved %d IP(s)", len(sysIPs)),
		Subitems: sub,
	}
}

// publicResolver returns a net.Resolver that bypasses the system resolver and
// queries Google's 8.8.8.8 directly. Used to disambiguate "user's DNS is broken"
// from "the host genuinely doesn't resolve."
func publicResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
}

func joinIPs(ips []net.IPAddr) string {
	parts := make([]string, len(ips))
	for i, ip := range ips {
		parts[i] = ip.IP.String()
	}
	return strings.Join(parts, ", ")
}

func ipSetsOverlap(a, b []net.IPAddr) bool {
	set := make(map[string]struct{}, len(a))
	for _, ip := range a {
		set[ip.IP.String()] = struct{}{}
	}
	for _, ip := range b {
		if _, ok := set[ip.IP.String()]; ok {
			return true
		}
	}
	return false
}

func checkTCP(ctx context.Context, host, port string) Result {
	if host == "" {
		return fail(nil, "no host")
	}

	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(dialCtx, host)
	if err != nil || len(ips) == 0 {
		// Fall back to dialing the hostname directly so the system resolver gets a turn.
		conn, derr := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(dialCtx, "tcp", net.JoinHostPort(host, port))
		if derr != nil {
			return fail(derr, fmt.Sprintf("connect to %s:%s failed: %v", host, port, derr))
		}
		_ = conn.Close()
		return ok(fmt.Sprintf("connected to %s:%s", host, port))
	}

	// Split resolved IPs into IPv4 and IPv6 so we can probe each family
	// independently — broken IPv6 routing is a common silent stall (AAAA resolves,
	// dial hangs until v4 fallback, on flaky networks the timeout fires first).
	var v4, v6 []net.IPAddr
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			v4 = append(v4, ip)
		} else {
			v6 = append(v6, ip)
		}
	}

	v4Res := dialFamily(dialCtx, v4, port)
	v6Res := dialFamily(dialCtx, v6, port)

	sub := make([]string, 0, 2)
	if v4Res.tried {
		sub = append(sub, fmt.Sprintf("IPv4: %s", v4Res.line))
	}
	if v6Res.tried {
		sub = append(sub, fmt.Sprintf("IPv6: %s", v6Res.line))
	}

	switch {
	case v4Res.ok || v6Res.ok:
		// At least one family connects — the gRPC client will use whichever
		// works. Surface a warning subitem if the other family failed silently.
		extras := map[string]string{}
		if v4Res.tried && !v4Res.ok {
			extras["v4_broken"] = "1"
		}
		if v6Res.tried && !v6Res.ok {
			extras["v6_broken"] = "1"
		}
		var detail string
		switch {
		case v4Res.ok && v6Res.ok:
			detail = "connected via IPv4 and IPv6"
		case v4Res.ok:
			detail = "connected via IPv4"
		default:
			detail = "connected via IPv6"
		}
		return Result{Pass: true, Detail: detail, Subitems: sub, Extras: extras}
	default:
		return Result{
			Pass:     false,
			Err:      v4Res.err,
			Detail:   fmt.Sprintf("could not connect to %s:%s on any resolved IP", host, port),
			Subitems: sub,
		}
	}
}

type familyResult struct {
	tried bool
	ok    bool
	line  string
	err   error
}

func dialFamily(ctx context.Context, ips []net.IPAddr, port string) familyResult {
	if len(ips) == 0 {
		return familyResult{tried: false, line: "no addresses returned by DNS"}
	}
	var lastErr error
	for _, ip := range ips {
		addr := net.JoinHostPort(ip.IP.String(), port)
		start := time.Now()
		conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", addr)
		if err == nil {
			elapsed := time.Since(start).Round(time.Millisecond)
			_ = conn.Close()
			return familyResult{
				tried: true,
				ok:    true,
				line:  fmt.Sprintf("connected to %s in %s", addr, elapsed),
			}
		}
		lastErr = err
	}
	return familyResult{
		tried: true,
		ok:    false,
		line:  fmt.Sprintf("failed: %v", lastErr),
		err:   lastErr,
	}
}

func checkTLS(ctx context.Context, host, port string, baseConfig *tls.Config, insecure bool) Result {
	if host == "" {
		return fail(nil, "no host")
	}
	cfg := baseConfig.Clone()
	if cfg.ServerName == "" {
		cfg.ServerName = host
	}
	// gRPC requires HTTP/2 — request "h2" via ALPN so we can verify the server
	// (and any middleboxes) negotiate it correctly. Without this, some corporate
	// proxies fall back to HTTP/1.1 silently and gRPC dials hang.
	cfg.NextProtos = []string{"h2"}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rawConn, err := dialer.DialContext(tlsCtx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return fail(err, fmt.Sprintf("tcp dial for TLS failed: %v", err))
	}
	tlsConn := tls.Client(rawConn, cfg)
	defer func() { _ = tlsConn.Close() }()

	if err := tlsConn.HandshakeContext(tlsCtx); err != nil {
		extras := map[string]string{}
		msg := err.Error()
		// macOS uses Apple's verifier which says "certificate is not trusted";
		// Linux/Windows use Go's verifier which says "unknown authority".
		if strings.Contains(msg, "unknown authority") || strings.Contains(msg, "not trusted") {
			extras["tls_unknown_authority"] = "1"
		}
		if strings.Contains(msg, "expired") || strings.Contains(msg, "not yet valid") {
			extras["tls_cert_time"] = "1"
		}
		var hostErr x509.HostnameError
		if asHostnameError(err, &hostErr) {
			extras["tls_hostname_mismatch"] = "1"
		}
		return Result{
			Pass:   false,
			Err:    err,
			Detail: fmt.Sprintf("handshake failed: %v", err),
			Extras: extras,
		}
	}

	state := tlsConn.ConnectionState()
	alpn := state.NegotiatedProtocol
	if alpn == "" {
		alpn = "(none)"
	}
	sub := []string{
		fmt.Sprintf("version: %s", tlsVersionString(state.Version)),
		fmt.Sprintf("cipher:  %s", tls.CipherSuiteName(state.CipherSuite)),
		fmt.Sprintf("alpn:    %s", alpn),
	}
	if len(state.PeerCertificates) > 0 {
		leaf := state.PeerCertificates[0]
		sub = append(sub,
			fmt.Sprintf("subject: %s", leaf.Subject.CommonName),
			fmt.Sprintf("issuer:  %s", leaf.Issuer.CommonName),
			fmt.Sprintf("expires: %s", leaf.NotAfter.Format(time.RFC3339)),
		)
	}
	extras := map[string]string{
		"insecure": boolStr(insecure),
		"alpn":     state.NegotiatedProtocol,
	}
	// gRPC requires h2 — if the middlebox stripped or downgraded ALPN, gRPC dials
	// will hang or fail mysteriously. Treat as a warning, not a failure: the
	// handshake itself worked, but downstream gRPC checks will likely fail.
	pass := true
	detail := "handshake succeeded"
	if insecure {
		detail += " (verification disabled)"
	}
	if state.NegotiatedProtocol != "h2" {
		pass = false
		detail = fmt.Sprintf("handshake succeeded but ALPN negotiated %q, not h2 — gRPC requires HTTP/2", alpn)
		extras["alpn_not_h2"] = "1"
	}
	return Result{
		Pass:     pass,
		Detail:   detail,
		Subitems: sub,
		Extras:   extras,
	}
}

func asHostnameError(err error, target *x509.HostnameError) bool {
	for e := err; e != nil; {
		if h, ok := e.(x509.HostnameError); ok {
			*target = h
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}

func checkHTTPS(ctx context.Context, url string) Result {
	httpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodHead, url, nil)
	if err != nil {
		return fail(err, fmt.Sprintf("could not build request: %v", err))
	}
	req.Header.Set("User-Agent", "tdl-diagnostics")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fail(err, fmt.Sprintf("request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	dateHeader := resp.Header.Get("Date")
	extras := map[string]string{}
	if dateHeader != "" {
		extras["date_header"] = dateHeader
	}

	res := Result{
		Pass:   resp.StatusCode < 400,
		Detail: fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
		Extras: extras,
	}
	if !res.Pass {
		res.Err = fmt.Errorf("status %d", resp.StatusCode)
	}
	return res
}

func checkGRPCPing(
	ctx context.Context,
	build func(addr, region string, insecure bool) (genproto.TrainingsClient, func() error, error),
	addr, region string,
	insecure bool,
) Result {
	client, closer, err := build(addr, region, insecure)
	if err != nil {
		return fail(err, fmt.Sprintf("dial failed: %v", err))
	}
	defer func() { _ = closer() }()

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := client.Ping(pingCtx, &emptypb.Empty{}); err != nil {
		return fail(err, fmt.Sprintf("Ping failed: %v", err))
	}
	return ok("Ping succeeded")
}

// checkGetTrainings runs an authenticated RPC (the same call powering `tdl training list`).
// It exercises the full path: dial + TLS + token metadata + server-side auth + response decoding.
// Skipped when the user is not configured — without a token there's nothing to authenticate.
func checkGetTrainings(
	ctx context.Context,
	build func(addr, region string, insecure bool) (genproto.TrainingsClient, func() error, error),
	addr, region string,
	insecure, configured bool,
) Result {
	if !configured {
		return Result{Skipped: true, Detail: "skipped: not configured (no token)"}
	}
	client, closer, err := build(addr, region, insecure)
	if err != nil {
		return fail(err, fmt.Sprintf("dial failed: %v", err))
	}
	defer func() { _ = closer() }()

	rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.GetTrainings(rpcCtx, &emptypb.Empty{})
	if err != nil {
		return fail(err, fmt.Sprintf("GetTrainings failed: %v", err))
	}
	return ok(fmt.Sprintf("got %d training(s)", len(resp.GetTrainings())))
}

func checkLatency(
	ctx context.Context,
	build func(addr, region string, insecure bool) (genproto.TrainingsClient, func() error, error),
	addr, region string,
	insecure bool,
) Result {
	const n = 5
	client, closer, err := build(addr, region, insecure)
	if err != nil {
		return fail(err, fmt.Sprintf("dial failed: %v", err))
	}
	defer func() { _ = closer() }()

	samples := make([]time.Duration, 0, n)
	fails := 0
	var lastErr error

	for i := range n {
		if i > 0 {
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return fail(ctx.Err(), "cancelled mid-probe")
			}
		}
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		start := time.Now()
		_, err := client.Ping(pingCtx, &emptypb.Empty{})
		cancel()
		if err != nil {
			fails++
			lastErr = err
			continue
		}
		samples = append(samples, time.Since(start))
	}

	if fails == n {
		return fail(lastErr, fmt.Sprintf("all %d pings failed; last error: %v", n, lastErr))
	}

	minD, maxD, sum := samples[0], samples[0], time.Duration(0)
	for _, s := range samples {
		if s < minD {
			minD = s
		}
		if s > maxD {
			maxD = s
		}
		sum += s
	}
	avg := sum / time.Duration(len(samples))

	extras := map[string]string{
		"loss":  fmt.Sprintf("%d", fails),
		"total": fmt.Sprintf("%d", n),
	}
	detail := fmt.Sprintf("min=%s avg=%s max=%s loss=%d/%d",
		minD.Round(time.Millisecond),
		avg.Round(time.Millisecond),
		maxD.Round(time.Millisecond),
		fails, n,
	)
	return Result{
		Pass:   fails == 0,
		Detail: detail,
		Extras: extras,
		Err:    lastErr,
	}
}

// checkStreaming opens a server-streaming RPC (VerifyExercise) with an
// intentionally invalid exercise_id and waits for the server's first response.
// Any clean response — including gRPC errors like NotFound or InvalidArgument —
// proves the streaming path is healthy. Only DeadlineExceeded indicates a real
// streaming-path problem (h2 flow control, idle proxy timeouts, etc.).
//
// Skipped when the user is not configured: the server gates the call on auth
// and we have no token to send.
func checkStreaming(
	ctx context.Context,
	build func(addr, region string, insecure bool) (genproto.TrainingsClient, func() error, error),
	addr, region string,
	insecure, configured bool,
	token string,
) Result {
	if !configured {
		return Result{Skipped: true, Detail: "skipped: not configured (no token)"}
	}
	client, closer, err := build(addr, region, insecure)
	if err != nil {
		return fail(err, fmt.Sprintf("dial failed: %v", err))
	}
	defer func() { _ = closer() }()

	streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Token goes in the request body to match how the production VerifyExercise
	// flow authenticates. With a valid token, the server can identify the user
	// and reject only on the bogus exercise_id (cleanest signal).
	stream, err := client.VerifyExercise(streamCtx, &genproto.VerifyExerciseRequest{
		ExerciseId: "tdl-diagnostics-probe",
		Token:      token,
	})
	if err != nil {
		return fail(err, fmt.Sprintf("stream open failed: %v", err))
	}

	_, err = stream.Recv()
	switch {
	case err == nil:
		// Surprising — server actually returned a verification response. The
		// stream works; that's all we need to know.
		return ok("stream opened and received response")
	case errorIsDeadlineExceeded(err):
		// The smoking-gun symptom. h2 flow control or proxy idle-timeout class.
		return Result{
			Pass:   false,
			Err:    err,
			Detail: "streaming RPC timed out waiting for server response — likely h2 / proxy issue",
			Extras: map[string]string{"stream_deadline": "1"},
		}
	default:
		// Any other gRPC error means the stream opened, the server processed
		// the request, and replied. From the diagnostics perspective: success.
		return Result{
			Pass:     true,
			Detail:   fmt.Sprintf("stream opened (server responded with %v)", err),
			Subitems: []string{fmt.Sprintf("server response: %v", err)},
		}
	}
}

func errorIsDeadlineExceeded(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return status.Code(err) == codes.DeadlineExceeded
}

// checkClockSkew compares local time to the Date: header captured by checkHTTPS.
// Skips if the HTTPS check did not run or did not return a Date header.
func checkClockSkew(httpsRes Result) Result {
	if httpsRes.Extras == nil {
		return Result{Skipped: true, Detail: "no HTTPS Date header to compare"}
	}
	dateHeader, ok := httpsRes.Extras["date_header"]
	if !ok || dateHeader == "" {
		return Result{Skipped: true, Detail: "no HTTPS Date header to compare"}
	}
	serverTime, err := http.ParseTime(dateHeader)
	if err != nil {
		return Result{Skipped: true, Detail: fmt.Sprintf("could not parse Date header: %v", err)}
	}
	drift := time.Since(serverTime)
	abs := drift
	if abs < 0 {
		abs = -abs
	}
	if abs > 60*time.Second {
		return fail(nil, fmt.Sprintf("local clock differs from server by %s — can break TLS", drift.Round(time.Second)))
	}
	return Result{Pass: true, Detail: fmt.Sprintf("within %s of server", abs.Round(time.Second))}
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// tunnelPrefixes is a heuristic list of interface-name prefixes that strongly
// suggest a VPN or tunnel — matched case-insensitively. Not exhaustive, but
// catches the common ones across macOS / Linux / Windows.
var tunnelPrefixes = []string{
	"utun",      // macOS WireGuard, OpenVPN, IPsec
	"tun",       // generic Linux/Unix tunnel (OpenVPN etc.)
	"tap",       // OpenVPN tap mode
	"ppp",       // PPTP / L2TP
	"tailscale", // Tailscale
	"wg",        // WireGuard on Linux
	"gpd",       // Cisco AnyConnect
	"zttap",     // ZeroTier
	"nordlynx",  // NordVPN WireGuard
}

func detectTunnels() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 {
			continue
		}
		if ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		nameLower := strings.ToLower(ifc.Name)
		matches := false
		for _, p := range tunnelPrefixes {
			if strings.HasPrefix(nameLower, p) {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		// macOS keeps many utun stubs around for Continuity / AirDrop / etc.
		// even with no VPN active. Require a non-link-local global address —
		// that's what an actual tunnel routes traffic over.
		if hasGlobalAddress(ifc) {
			out = append(out, ifc.Name)
		}
	}
	return out
}

func hasGlobalAddress(ifc net.Interface) bool {
	addrs, err := ifc.Addrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		if ip.IsUnspecified() {
			continue
		}
		return true
	}
	return false
}
