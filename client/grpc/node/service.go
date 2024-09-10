package node

import (
	context "context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/atomone-hub/atomone/client"
	sdk "github.com/atomone-hub/atomone/types"
)

// RegisterNodeService registers the node gRPC service on the provided gRPC router.
func RegisterNodeService(clientCtx client.Context, server gogogrpc.Server) {
	RegisterServiceServer(server, NewQueryServer(clientCtx))
}

// RegisterGRPCGatewayRoutes mounts the node gRPC service's GRPC-gateway routes
// on the given mux object.
func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
	_ = RegisterServiceHandlerClient(context.Background(), mux, NewServiceClient(clientConn))
}

var _ ServiceServer = queryServer{}

type queryServer struct {
	clientCtx client.Context
}

func NewQueryServer(clientCtx client.Context) ServiceServer {
	return queryServer{
		clientCtx: clientCtx,
	}
}

func (s queryServer) Config(ctx context.Context, _ *ConfigRequest) (*ConfigResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	return &ConfigResponse{
		MinimumGasPrice: sdkCtx.MinGasPrices().String(),
	}, nil
}
