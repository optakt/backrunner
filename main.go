package main

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func main() {

	var (
		logLevel string
		apiURL   string
	)

	pflag.StringVarP(&logLevel, "log-level", "l", "debug", "Zerolog logger minimum severity level")
	pflag.StringVarP(&apiURL, "api-url", "a", "wss://eth-mainnet.g.alchemy.com/v2/6ISgOiZx8jxDGwr_OrR7-ulc6q0lVsR6", "JSON RPC API URL")

	pflag.Parse()

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Str("log_level", logLevel).Err(err).Msg("invalid log level")
	}
	log = log.Level(level)

	rpcClient, err := rpc.Dial(apiURL)
	if err != nil {
		log.Fatal().Str("api_url", apiURL).Err(err).Msg("could not create RPC connection")
	}

	client := gethclient.New(rpcClient)

	txHashes := make(chan common.Hash, 10)
	sub, err := client.SubscribePendingTransactions(context.Background(), txHashes)
	if err != nil {
		log.Fatal().Err(err).Msg("could not subscribe to pending transactions")
	}

	_ = sub

	for txHash := range txHashes {
		log.Debug().Hex("tx_hash", txHash[:]).Msg("pending transaction received")
	}

	os.Exit(0)
}
