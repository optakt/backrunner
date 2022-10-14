package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/ethclient"
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

	client, err := ethclient.Dial(apiURL)
	if err != nil {
		log.Fatal().Str("api_url", apiURL).Err(err).Msg("could not connect to JSON RPC API")
	}

	os.Exit(0)
}
