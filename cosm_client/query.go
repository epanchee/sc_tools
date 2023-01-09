package cosm_client

import (
	"context"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/pkg/errors"
)

func NewQueryClient(rpcEndpoint string) (types.QueryClient, error) {
	options := []cosmosclient.Option{
		cosmosclient.WithNodeAddress(rpcEndpoint),
	}
	client, err := cosmosclient.New(context.Background(), options...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create query client")
	}

	return types.NewQueryClient(client.Context()), nil
}
