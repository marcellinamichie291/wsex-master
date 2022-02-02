package gateio

import "github.com/shiguantian/wsex"

type Gate struct {
	GateRest
	GateWs
}

func New(options wsex.Options) *Gate {
	instance := &Gate{}
	instance.GateRest.Init(options)
	instance.GateWs.Init(options)

	if len(options.Markets) == 0 {
		instance.GateWs.Option.Markets, _ = instance.FetchMarkets()
	}
	return instance
}
