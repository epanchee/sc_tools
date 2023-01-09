package cmd

import (
	"context"
	"fmt"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/sha3"
	"sc-tools/cosm_client"
	"strconv"
)

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		} else {
			panic(fmt.Sprint(err, "failed to read config file"))
		}
	}

	log.Info("Using config file: ", viper.ConfigFileUsed())
}

func runDeploy(cmd *cobra.Command, _ []string) error {
	node := viper.GetString("node")
	if !viper.InConfig("node") {
		node = cmd.Flags().Lookup("node").Value.String()
	}
	prefix := viper.GetString("prefix")
	if !viper.InConfig("prefix") {
		prefix = cmd.Flags().Lookup("prefix").Value.String()
	}
	coinType := viper.GetUint32("coin_type")
	if !viper.InConfig("coin_type") {
		ct, err := cmd.Flags().GetUint32("coin-type")
		if err != nil {
			return errors.Wrapf(err, "failed to get coin-type flag")
		}
		coinType = ct
	}
	patched := viper.GetBool("patched_optimizer")
	if !viper.InConfig("patched_optimizer") {
		p, err := cmd.Flags().GetBool("patched-optimizer")
		if err != nil {
			return errors.Wrapf(err, "failed to get patched-optimizer flag")
		}
		patched = p
	}
	image := viper.GetString("optimizer_image")
	if !viper.InConfig("optimizer_image") {
		image = cmd.Flags().Lookup("optimizer-image").Value.String()
	}

	flagsMap := map[string]string{
		"commit-link": cmd.Flags().Lookup("commit-link").Value.String(),
		"crate-name":  cmd.Flags().Lookup("crate-name").Value.String(),
		"node":        node,
		"prefix":      prefix,
		"image":       image,
	}

	onlyBuild, err := cmd.Flags().GetBool("only-build")
	if err != nil {
		return errors.Wrapf(err, "failed to get only_build flag")
	}

	client, err := cosm_client.NewSigningClient(
		flagsMap["node"], flagsMap["prefix"], coinType,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create client")
	} else {
		log.Info("Client created with signer address: ", client.SignerAcc.Address(prefix))
	}

	allowArm, err := cmd.Flags().GetBool("allow-arm")
	if err != nil {
		return errors.Wrapf(err, "failed to get allow-arm flag")
	}
	wasmByteCode, err := checkoutAndBuild(flagsMap, allowArm, patched)
	if err != nil {
		return err
	} else {
		log.Info("Wasm code was successfully built")
	}

	if !onlyBuild {
		if err = client.StoreCode(wasmByteCode); err != nil {
			return err
		}
	}

	return nil
}

func runCheck(cmd *cobra.Command, args []string) error {
	codeId, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return errors.Wrapf(err, "failed to parse code id")
	}

	node := viper.GetString("node")
	if !viper.InConfig("node") {
		node = cmd.Flags().Lookup("node").Value.String()
	}

	image := viper.GetString("optimizer_image")
	if !viper.InConfig("optimizer_image") {
		image = cmd.Flags().Lookup("optimizer-image").Value.String()
	}

	flagsMap := map[string]string{
		"commit-link": cmd.Flags().Lookup("commit-link").Value.String(),
		"crate-name":  cmd.Flags().Lookup("crate-name").Value.String(),
		"node":        node,
		"image":       image,
	}

	client, err := cosm_client.NewQueryClient(flagsMap["node"])
	if err != nil {
		return err
	}

	allowArm, err := cmd.Flags().GetBool("allow-arm")
	if err != nil {
		return errors.Wrapf(err, "failed to get allow-arm flag")
	}

	patched := viper.GetBool("patched_optimizer")
	if !viper.InConfig("patched_optimizer") {
		p, err := cmd.Flags().GetBool("patched-optimizer")
		if err != nil {
			return errors.Wrapf(err, "failed to get patched-optimizer flag")
		}
		patched = p
	}

	wasmByteCode, err := checkoutAndBuild(flagsMap, allowArm, patched)
	if err != nil {
		return err
	} else {
		log.Info("Wasm code was successfully built")
	}
	builtSha := sha3.Sum256(wasmByteCode)

	resp, err := client.Code(context.Background(), &wasm.QueryCodeRequest{
		CodeId: codeId,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to query codeId")
	}
	var chainHash [32]byte
	copy(chainHash[:], resp.DataHash.Bytes())

	fmt.Printf("Chain hash: %x\n", chainHash)
	fmt.Printf("Built hash: %x\n", builtSha)

	if builtSha != chainHash {
		log.Error("Wasm code hashes mismatch")
	} else {
		log.Info("Wasm code hashes are equal")
	}

	return nil
}

var (
	rootCmd = &cobra.Command{
		Use:   "sc-tools",
		Short: "Smart contracts builder and deployer",
		Long: `Smart contracts builder and deployer uses rust-optimizer docker images to build wasm files. 
Then it uploads them to the chain.`,
	}
	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Build and deploy wasm binary",
		Long:  `Compile smart contract into wasm and deploy it in chain`,
		RunE:  runDeploy,
	}
	checkCmd = &cobra.Command{
		Use:   "check [code id]",
		Short: "Build and check wasm binary",
		Long:  `Build and check wasm binary against codeID in chain"`,
		RunE:  runCheck,
		Args:  cobra.ExactArgs(1),
	}
)

// Execute executes the root command.
func Execute() error {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("commit-link", "l", "", "Full link to commit")
	rootCmd.PersistentFlags().StringP("crate-name", "c", "", "Name of the crate with the contract")
	_ = rootCmd.MarkPersistentFlagRequired("commit-link")
	_ = rootCmd.MarkPersistentFlagRequired("crate-name")

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(checkCmd)

	deployCmd.Flags().Bool("only-build", false, "Only build wasm")
	deployCmd.Flags().Bool("allow-arm", false, "Allow arm compilation")
	deployCmd.Flags().Bool("patched-optimizer", false, "Use patched optimizer for faster compilation")
	deployCmd.Flags().String("optimizer-image", "cosmwasm/workspace-optimizer:0.12.11", "Rust optimizer image")
	deployCmd.MarkFlagsMutuallyExclusive("optimizer-image", "patched-optimizer")

	checkCmd.Flags().Bool("allow-arm", false, "Allow arm compilation")
	checkCmd.Flags().Bool("patched-optimizer", false, "Use patched optimizer for faster compilation")
	checkCmd.Flags().String("optimizer-image", "cosmwasm/workspace-optimizer:0.12.11", "Rust optimizer image")
	checkCmd.MarkFlagsMutuallyExclusive("optimizer-image", "patched-optimizer")

	rootCmd.PersistentFlags().StringP("node", "n", "", "Node address")
	rootCmd.PersistentFlags().StringP("prefix", "p", "", "Address prefix")
	rootCmd.PersistentFlags().Uint32("coin-type", 330, "Coin type for HD path derivation")

	return rootCmd.Execute()
}
