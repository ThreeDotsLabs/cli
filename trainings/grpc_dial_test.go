package trainings

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

// Some resolvers (seen behind VPNs) silently drop TXT queries. grpc-go's dns resolver
// looks up the _grpc_config TXT record before the first call can pick an address, so
// without service config disabled the first call times out.
func TestGrpcClient_resolver_dropping_TXT_queries(t *testing.T) {
	serverPort := startPingServer(t)
	dnsAddr := startDNSServerDroppingTXT(t)

	target := fmt.Sprintf("dns://%s/localhost:%d", dnsAddr, serverPort)
	client, closeClient, err := (&Handlers{}).newFreshGrpcClient(target, "", true)
	require.NoError(t, err)
	t.Cleanup(func() { _ = closeClient() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = client.Ping(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

type pingOnlyServer struct {
	genproto.UnimplementedTrainingsServer
}

func (pingOnlyServer) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func startPingServer(t *testing.T) int {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer(grpc.Creds(credentials.NewServerTLSFromCert(selfSignedCert(t))))
	genproto.RegisterTrainingsServer(srv, pingOnlyServer{})

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return lis.Addr().(*net.TCPAddr).Port
}

func selfSignedCert(t *testing.T) *tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// startDNSServerDroppingTXT answers A queries with 127.0.0.1, AAAA queries with no
// records, and never answers TXT queries.
func startDNSServerDroppingTXT(t *testing.T) string {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}

			resp, ok := dnsResponse(buf[:n])
			if !ok {
				continue
			}
			_, _ = conn.WriteTo(resp, addr)
		}
	}()

	return conn.LocalAddr().String()
}

func dnsResponse(query []byte) ([]byte, bool) {
	var req dnsmessage.Message
	if err := req.Unpack(query); err != nil || len(req.Questions) != 1 {
		return nil, false
	}
	q := req.Questions[0]

	resp := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:                 req.ID,
			Response:           true,
			Authoritative:      true,
			RecursionDesired:   req.RecursionDesired,
			RecursionAvailable: true,
		},
		Questions: req.Questions,
	}

	switch q.Type {
	case dnsmessage.TypeA:
		resp.Answers = []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 60},
			Body:   &dnsmessage.AResource{A: [4]byte{127, 0, 0, 1}},
		}}
	case dnsmessage.TypeAAAA:
		// No records.
	default:
		// Drop TXT (and anything else) without answering.
		return nil, false
	}

	packed, err := resp.Pack()
	if err != nil {
		return nil, false
	}
	return packed, true
}
