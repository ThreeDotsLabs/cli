package diagnostics

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInterpret(t *testing.T) {
	pass := func(name string) Result { return Result{Name: name, Pass: true} }
	fail := func(name string, err error) Result { return Result{Name: name, Pass: false, Err: err} }

	allPass := []Result{
		pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS), pass(NameH2),
		pass(NameHTTPS), pass(NamePing), pass(NameGetTrainings), pass(NameStream),
		pass(NameLat), pass(NameClock),
	}

	tests := []struct {
		name        string
		results     []Result
		snap        Snapshot
		contains    []string // headline phrases the interpretation must contain
		notContains []string // phrases that must NOT appear
	}{
		{
			name:     "all_pass",
			results:  allPass,
			snap:     Snapshot{Host: "example.com", Port: "443"},
			contains: []string{"All checks passed"},
		},
		{
			name: "dns_fail_short_circuits",
			results: []Result{
				pass(NameProxy),
				fail(NameDNS, errors.New("no such host")),
				fail(NameTCP, errors.New("would never run")),
				fail(NameTLS, errors.New("would never run")),
			},
			snap:     Snapshot{Host: "no.such.host", Port: "443"},
			contains: []string{"DNS resolution failed", "no.such.host"},
		},
		{
			name: "tcp_fail_with_dns_ok",
			results: []Result{
				pass(NameProxy), pass(NameDNS),
				fail(NameTCP, errors.New("connection refused")),
				fail(NameTLS, errors.New("would never run")),
			},
			snap:     Snapshot{Host: "example.com", Port: "443"},
			contains: []string{"Cannot reach example.com on port 443", "firewall"},
		},
		{
			name: "tls_unknown_authority",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: false, Err: errors.New("x509: certificate signed by unknown authority"),
					Extras: map[string]string{"tls_unknown_authority": "1"}},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"unknown authority", "TLS-interception", "--insecure"},
		},
		{
			name: "tls_cert_time",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: false, Err: errors.New("expired"),
					Extras: map[string]string{"tls_cert_time": "1"}},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"time-validity", "clock"},
		},
		{
			name: "grpc_unauthenticated",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS), pass(NameHTTPS),
				{Name: NamePing, Pass: false, Err: status.Error(codes.Unauthenticated, "bad token")},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"rejected your token", "training configure"},
		},
		{
			name: "grpc_deadline_exceeded",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS), pass(NameHTTPS),
				{Name: NamePing, Pass: false, Err: status.Error(codes.DeadlineExceeded, "")},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"Ping timed out", "unstable"},
		},
		{
			name: "latency_partial_loss",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS), pass(NameHTTPS),
				pass(NamePing),
				{Name: NameLat, Pass: false, Err: errors.New("timeout"),
					Extras: map[string]string{"loss": "2", "total": "5"}},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"unstable", "2 of 5 pings failed"},
		},
		{
			name: "proxy_note_appended_when_set",
			results: []Result{
				{Name: NameProxy, Pass: true, Extras: map[string]string{"proxy_set": "1"}},
				pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: false, Err: errors.New("oops"),
					Extras: map[string]string{"tls_unknown_authority": "1"}},
			},
			snap:     Snapshot{Host: "example.com", ProxyEnv: map[string]string{"HTTPS_PROXY": "http://corp.local:8080"}},
			contains: []string{"unknown authority", "proxy environment variables are set"},
		},
		{
			name: "alpn_not_h2",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: false, Err: errors.New("alpn"),
					Extras: map[string]string{"alpn_not_h2": "1", "alpn": "http/1.1"}},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"ALPN did not negotiate h2", "http/1.1", "HTTP/2"},
		},
		{
			name: "system_dns_broken_public_works",
			results: []Result{
				pass(NameProxy),
				{Name: NameDNS, Pass: false, Err: errors.New("system fail"),
					Extras: map[string]string{"system_dns_broken": "1"}},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"system DNS is broken", "8.8.8.8", "resolv.conf"},
		},
		{
			name: "streaming_deadline",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS),
				pass(NameHTTPS), pass(NamePing), pass(NameGetTrainings),
				{Name: NameStream, Pass: false, Err: errors.New("deadline"),
					Extras: map[string]string{"stream_deadline": "1"}},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"Streaming gRPC request timed out", "exercise verification", "HTTP/2 flow control"},
		},
		{
			name: "vpn_note_appended",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: false, Err: errors.New("oops"),
					Extras: map[string]string{"tls_unknown_authority": "1"}},
			},
			snap:     Snapshot{Host: "example.com", Tunnels: []string{"utun4"}},
			contains: []string{"unknown authority", "VPN / tunnel interface(s) active", "utun4"},
		},
		{
			name: "pq_kx_note_appended_on_grpc_fail",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: true,
					Extras: map[string]string{"curve": "X25519MLKEM768", "curve_pq": "1", "alpn": "h2"}},
				pass(NameHTTPS),
				{Name: NamePing, Pass: false, Err: status.Error(codes.DeadlineExceeded, "")},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"Ping timed out", "X25519MLKEM768", "GODEBUG", "tlsmlkem=0", "PowerShell"},
		},
		{
			name: "h2_probe_no_frame_is_the_headline",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS),
				{Name: NameH2, Pass: false, Err: errors.New("read frame: i/o timeout"),
					Extras: map[string]string{"h2_no_frame": "1"}},
				pass(NameHTTPS),
				{Name: NamePing, Pass: false, Err: status.Error(codes.DeadlineExceeded, "")},
			},
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"HTTP/2 traffic is being blocked", "TLS handshake completes", "personal hotspot"},
		},
		{
			name: "multiple_grpc_deadline_with_tls_pass_is_h2_middlebox",
			results: []Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				pass(NameTLS), pass(NameH2), pass(NameHTTPS),
				{Name: NamePing, Pass: false, Err: status.Error(codes.DeadlineExceeded, "")},
				{Name: NameGetTrainings, Pass: false, Err: status.Error(codes.DeadlineExceeded, "")},
				{Name: NameStream, Pass: false, Err: errors.New("deadline"),
					Extras: map[string]string{"stream_deadline": "1"}},
			},
			snap: Snapshot{Host: "example.com"},
			// must NOT match the single-Ping headline; must match the combined one
			contains:    []string{"Every gRPC call timed out", "fingerprint of HTTP/2", "personal hotspot"},
			notContains: []string{"Ping timed out.", "see latency probe"},
		},
		{
			name: "clock_skew_warning_surfaced",
			results: append([]Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP), pass(NameTLS), pass(NameH2),
				pass(NameHTTPS), pass(NamePing), pass(NameGetTrainings), pass(NameStream),
				pass(NameLat),
			}, Result{Name: NameClock, Pass: true,
				Extras: map[string]string{"clock_warn": "1", "drift_sec": "50"}}),
			snap:     Snapshot{Host: "example.com"},
			contains: []string{"All checks passed", "system clock is 50s off", "NTP-synced"},
		},
		{
			name: "pq_kx_note_NOT_appended_when_all_grpc_pass",
			results: append([]Result{
				pass(NameProxy), pass(NameDNS), pass(NameTCP),
				{Name: NameTLS, Pass: true,
					Extras: map[string]string{"curve": "X25519MLKEM768", "curve_pq": "1", "alpn": "h2"}},
				pass(NameH2),
			}, pass(NameHTTPS), pass(NamePing), pass(NameGetTrainings), pass(NameStream),
				pass(NameLat), pass(NameClock)),
			snap:        Snapshot{Host: "example.com"},
			contains:    []string{"All checks passed"},
			notContains: []string{"tlsmlkem", "GODEBUG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(Interpret(tt.results, tt.snap))
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Interpret() output missing %q\n--- got ---\n%s", want, got)
				}
			}
			for _, unwanted := range tt.notContains {
				if strings.Contains(got, unwanted) {
					t.Errorf("Interpret() output should NOT contain %q\n--- got ---\n%s", unwanted, got)
				}
			}
		})
	}
}

// stripANSI removes ANSI color escapes so test assertions don't need to match them.
func stripANSI(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in:
			if r == 'm' {
				in = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestTokenStatus(t *testing.T) {
	cases := map[string]string{
		"":        "missing",
		"abc":     "present (length=3)",
		"  abc  ": "present (length=7, has leading/trailing whitespace)",
		"abc\n":   "present (length=4, has leading/trailing whitespace)",
	}
	for in, want := range cases {
		if got := tokenStatus(in); got != want {
			t.Errorf("tokenStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitHostPort(t *testing.T) {
	cases := []struct {
		in         string
		host, port string
	}{
		{"academy-grpc.threedots.tech:443", "academy-grpc.threedots.tech", "443"},
		{"eu.academy-grpc.threedots.tech:443", "eu.academy-grpc.threedots.tech", "443"},
		{"justhost", "justhost", "443"},
	}
	for _, c := range cases {
		h, p := splitHostPort(c.in)
		if h != c.host || p != c.port {
			t.Errorf("splitHostPort(%q) = (%q,%q), want (%q,%q)", c.in, h, p, c.host, c.port)
		}
	}
}
