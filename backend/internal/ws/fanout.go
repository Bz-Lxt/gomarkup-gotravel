package ws

import "time"

// FanoutPolicy documents the handwritten elasticity rules used by Hub/Client.
type FanoutPolicy struct {
	CoalesceWindow   time.Duration
	SendBuffer       int
	PrioBuffer       int
	InboundRate      float64
	InboundBurst     float64
	DropOldest       bool
	MaxConsecutiveDrop int
	SlowWriteTimeout   time.Duration
	MaxSlowWrites      int
}

func DefaultFanoutPolicy() FanoutPolicy {
	return FanoutPolicy{
		CoalesceWindow:     200 * time.Millisecond,
		SendBuffer:         sendBuf,
		PrioBuffer:         prioBuf,
		InboundRate:        5,
		InboundBurst:       10,
		DropOldest:         true,
		MaxConsecutiveDrop: maxDropRun,
		SlowWriteTimeout:   writeWait,
		MaxSlowWrites:      maxSlowFail,
	}
}

// CompressRatio compares naive N×N broadcast count vs coalesced outbound count.
func CompressRatio(naive, outbound int64) float64 {
	if naive <= 0 {
		return 0
	}
	r := 1 - float64(outbound)/float64(naive)
	if r < 0 {
		return 0
	}
	return r
}
