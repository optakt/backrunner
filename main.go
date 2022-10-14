package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	SwapRouter2 = "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
)

func main() {

	var (
		logLevel string
		apiURL   string
	)

	pflag.StringVarP(&logLevel, "log-level", "l", "info", "Zerolog logger minimum severity level")
	pflag.StringVarP(&apiURL, "api-url", "a", "wss://eth-mainnet.g.alchemy.com/v2/6ISgOiZx8jxDGwr_OrR7-ulc6q0lVsR6", "JSON RPC API URL")

	pflag.Parse()

	ctx := context.Background()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Str("log_level", logLevel).Err(err).Msg("invalid log level")
	}
	log = log.Level(level)

	router, err := abi.JSON(strings.NewReader(Router2))
	if err != nil {
		log.Fatal().Err(err).Msg("could not decode router ABI")
	}

	multicall, err := abi.JSON(strings.NewReader(Multicall2))
	if err != nil {
		log.Fatal().Err(err).Msg("could not decode multicall ABI")
	}

	rpcClient, err := rpc.Dial(apiURL)
	if err != nil {
		log.Fatal().Str("api_url", apiURL).Err(err).Msg("could not create RPC connection")
	}

	eth := ethclient.NewClient(rpcClient)
	geth := gethclient.New(rpcClient)

	txHashes := make(chan common.Hash, 10)
	sub, err := geth.SubscribePendingTransactions(context.Background(), txHashes)
	if err != nil {
		log.Fatal().Err(err).Msg("could not subscribe to pending transactions")
	}

Loop:
	for {

		select {

		case <-sig:
			break Loop

		case err := <-sub.Err():
			log.Fatal().Err(err).Msg("encountered subscription error")

		case txHash := <-txHashes:

			tx, pending, err := eth.TransactionByHash(ctx, txHash)
			if err != nil {
				log.Error().Err(err).Msg("could not get transaction")
			}
			if !pending {
				log.Debug().Msg("tx no longer pending")
				continue
			}

			if tx.To() == nil {
				log.Debug().Msg("to address empty")
				continue
			}

			to := *tx.To()
			if to != common.HexToAddress(SwapRouter2) {
				log.Debug().Msg("skipping irrelevant to address")
				continue
			}

			log.Info().
				Hex("tx_hash", txHash[:]).
				Msg("found transaction on Uniswap v2 Router 2")
		}
	}

	os.Exit(0)
}
