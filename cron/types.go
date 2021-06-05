package cron

import (
	"github.com/PolkaFoundry/go-substrate-rpc-client/v3/types"
)

type BabeEpochConfiguration struct {
	C [2]types.U64
	AllowedSlots types.U32
}

type BlockNumber types.U32