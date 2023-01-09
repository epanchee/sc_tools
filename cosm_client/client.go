package cosm_client

import (
	"context"
	"fmt"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type ClientWrapper struct {
	client    cosmosclient.Client
	SignerAcc cosmosaccount.Account
}

func NewSigningClient(rpcEndpoint, addrPrefix string, coinType uint32) (ClientWrapper, error) {
	client, err := createClient(rpcEndpoint, addrPrefix, coinType)
	if err != nil {
		return ClientWrapper{}, err
	}

	mnemonic := viper.GetString("mnemonic")
	if err != nil {
		return ClientWrapper{}, err
	}
	signerAcc, err := client.AccountRegistry.Import("deployer", mnemonic, "")
	if err != nil {
		return ClientWrapper{}, err
	}

	return ClientWrapper{
		client:    client,
		SignerAcc: signerAcc,
	}, nil
}

func (c *ClientWrapper) SendTx(msg sdk.Msg) (cosmosclient.Response, error) {
	if resp, err := c.client.BroadcastTx("deployer", msg); err != nil {
		return cosmosclient.Response{}, err
	} else {
		return resp, nil
	}
}

func (c *ClientWrapper) StoreCode(wasmByteCode []byte) error {
	storeCodeMsg := wasm.MsgStoreCode{
		Sender:                c.SignerAcc.Address("terra"),
		WASMByteCode:          wasmByteCode,
		InstantiatePermission: nil,
	}
	log.Info("Storing code ...")
	if txResp, err := c.SendTx(&storeCodeMsg); err != nil {
		return errors.Wrapf(err, "failed to store code")
	} else {
		fmt.Println(txResp)
	}

	return nil
}

func createClient(rpcEndpoint, addrPrefix string, coinType uint32) (cosmosclient.Client, error) {
	sdk.GetConfig().SetCoinType(coinType)

	// Create a Cosmos client instance
	options := []cosmosclient.Option{
		cosmosclient.WithNodeAddress(rpcEndpoint),
		cosmosclient.WithAddressPrefix(addrPrefix),
		cosmosclient.WithKeyringBackend(cosmosaccount.KeyringMemory),
	}
	client, err := cosmosclient.New(context.Background(), options...)
	if err != nil {
		return cosmosclient.Client{}, errors.Wrapf(err, "failed to create client")
	}
	client.Factory = client.Factory.WithGasPrices("0.15uluna").WithFees("0.15uluna")

	return client, nil
}
