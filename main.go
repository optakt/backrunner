package main

import (
	"bytes"
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
	AddressRouter = "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45"
)

var (
	SigMulticall = []byte{0x5a, 0xe4, 0x01, 0xdc}
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

	router, err := abi.JSON(strings.NewReader(ABIRouter2))
	if err != nil {
		log.Fatal().Err(err).Msg("could not decode router ABI")
	}

	multicall, err := router.MethodById(SigMulticall)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get multicall function")
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

			tx, _, err := eth.TransactionByHash(ctx, txHash)
			if err != nil {
				log.Debug().Err(err).Msg("could not get transaction")
				continue
			}

			if tx.To() == nil {
				log.Debug().Msg("skipping contract creation")
				continue
			}

			to := *tx.To()
			if to != common.HexToAddress(AddressRouter) {
				log.Debug().Msg("skipping non-router transaction")
				continue
			}

			inputData := tx.Data()
			if !bytes.Equal(inputData[0:4], SigMulticall) {
				log.Debug().Msg("skipping non-multicall transaction")
				continue
			}

			values, err := multicall.Inputs.Unpack(inputData[4:])
			if err != nil {
				log.Error().Err(err).Msg("could not unpack multicall")
				continue
			}

			var input Multicall
			err = multicall.Inputs.Copy(&input, values)
			if err != nil {
				log.Error().Err(err).Msg("could not copy values")
				continue
			}

			log.Info().
				Hex("tx_hash", txHash[:]).
				Hex("input_data", inputData).
				Time("deadline", time.Unix(input.Deadline.Int64(), 0)).
				Msg("unpacked qualifying multicall")

			for _, callData := range input.Data {
				log.Info().
					Hex("call_data", callData).
					Msg("unwound multicall call")
			}
		}
	}

	sub.Unsubscribe()

	close(txHashes)
	for range txHashes {
	}

	eth.Close()

	os.Exit(0)
}
