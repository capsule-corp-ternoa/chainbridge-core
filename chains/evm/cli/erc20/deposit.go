package erc20

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/cli/flags"
	"github.com/ChainSafe/chainbridge-core/chains/evm/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/evmtransaction"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var depositCmd = &cobra.Command{
	Use:   "deposit",
	Short: "Initiate a transfer of ERC20 tokens",
	Long:  "Initiate a transfer of ERC20 tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		txFabric := evmtransaction.NewTransaction
		return DepositEVMCMD(cmd, args, txFabric)
	},
}

func BindERC20DepositCLIFlags(cli *cobra.Command) {
	cli.Flags().String("recipient", "", "address of recipient")
	cli.Flags().String("bridge", "", "address of bridge contract")
	cli.Flags().String("amount", "", "amount to deposit")
	cli.Flags().String("value", "0", "value of ETH that should be sent along with deposit to cover possible fees. In ETH (decimals are allowed)")
	cli.Flags().String("destId", "", "destination chain ID")
	cli.Flags().String("resourceId", "", "resource ID for transfer")
	cli.Flags().Uint64("decimals", 0, "ERC20 token decimals")
	cli.MarkFlagRequired("decimals")
}

func init() {
	BindERC20DepositCLIFlags(depositCmd)
}

func DepositEVMCMD(cmd *cobra.Command, args []string, txFabric calls.TxFabric) error {
	recipientAddress := cmd.Flag("recipient").Value.String()
	bridgeAddress := cmd.Flag("bridge").Value.String()
	amount := cmd.Flag("amount").Value.String()
	destinationId := cmd.Flag("destId").Value.String()
	resourceId := cmd.Flag("resourceId").Value.String()

	// fetch global flag values
	url, gasLimit, gasPrice, senderKeyPair, err := flags.GlobalFlagValues(cmd)
	if err != nil {
		return fmt.Errorf("could not get global flags: %v", err)
	}

	// ignore success bool
	decimals, _ := big.NewInt(0).SetString(cmd.Flag("decimals").Value.String(), 10)

	if !common.IsHexAddress(bridgeAddress) {
		return fmt.Errorf("invalid bridge address %s", bridgeAddress)
	}

	bridgeAddr := common.HexToAddress(bridgeAddress)

	if !common.IsHexAddress(recipientAddress) {
		return fmt.Errorf("invalid recipient address %s", recipientAddress)
	}
	recipientAddr := common.HexToAddress(recipientAddress)

	realAmount, err := calls.UserAmountToWei(amount, decimals)
	if err != nil {
		return err
	}

	resourceIDBytes := calls.SliceTo32Bytes(common.Hex2Bytes(resourceId))

	ethClient, err := evmclient.NewEVMClientFromParams(url, senderKeyPair.PrivateKey(), gasPrice)
	if err != nil {
		log.Error().Err(fmt.Errorf("eth client intialization error: %v", err))
		return err
	}

	destinationIdInt, err := strconv.Atoi(destinationId)
	if err != nil {
		log.Error().Err(fmt.Errorf("destination ID conversion error: %v", err))
		return err
	}

	// TODO: confirm correct arguments
	input, err := calls.PrepareErc20DepositInput(bridgeAddr, recipientAddr, realAmount, resourceIDBytes, uint8(destinationIdInt))
	if err != nil {
		log.Error().Err(fmt.Errorf("erc20 deposit input error: %v", err))
		return err
	}
	// destinationId
	txHash, err := calls.Transact(ethClient, txFabric, &recipientAddr, input, gasLimit)
	if err != nil {
		log.Error().Err(fmt.Errorf("erc20 deposit error: %v", err))
		return err
	}

	log.Debug().Msgf("erc20 deposit hash: %s", txHash.Hex())

	log.Info().Msgf("%s tokens were transferred to %s from %s", amount, recipientAddr.Hex(), senderKeyPair.CommonAddress().String())
	return nil
}