package trainings

import (
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
	"google.golang.org/grpc"
)

func NewGrpcClient(serverAddr string) genproto.ServerClient {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	return genproto.NewServerClient(conn)
}
