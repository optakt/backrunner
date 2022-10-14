package main

import (
	"math/big"
)

type Multicall struct {
	Deadline *big.Int `json:"deadline"`
	Data     [][]byte `json:"data"`
}
