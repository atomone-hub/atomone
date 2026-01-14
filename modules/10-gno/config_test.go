package gno_test

import (
	"time"

	ibcgno "github.com/atomone-hub/atomone/modules/10-gno"
)

type GnoConfig struct {
	TrustLevel      ibcgno.Fraction
	TrustingPeriod  time.Duration
	UnbondingPeriod time.Duration
	MaxClockDrift   time.Duration
}

const (
	TrustingPeriod  time.Duration = time.Hour * 24
	UnbondingPeriod time.Duration = time.Hour * 24 * 2
	MaxClockDrift   time.Duration = time.Second * 10
)

func NewGnoConfig() *GnoConfig {
	return &GnoConfig{
		TrustLevel:      ibcgno.DefaultTrustLevel,
		TrustingPeriod:  TrustingPeriod,
		UnbondingPeriod: UnbondingPeriod,
		MaxClockDrift:   MaxClockDrift,
	}
}

func (*GnoConfig) GetClientType() string {
	return ibcgno.Gno
}
