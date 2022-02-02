package binance

import (
	"github.com/shiguantian/wsex"
)

type BinanceFuture struct {
	BinanceFutureRest
	BinanceFutureWs
}

func NewFuture(options wsex.Options, futureOptions wsex.FutureOptions)  *BinanceFuture {
	instance := &BinanceFuture{}
	instance.BinanceFutureRest.Init(options)
	instance.BinanceFutureRest.accountType = futureOptions.FutureAccountType
	instance.BinanceFutureRest.contractType = futureOptions.ContractType
	instance.BinanceFutureRest.futuresKind = futureOptions.FuturesKind

	instance.BinanceFutureWs.Init(options)
	instance.BinanceFutureWs.accountType = futureOptions.FutureAccountType
	instance.BinanceFutureWs.contractType = futureOptions.ContractType
	instance.BinanceFutureWs.futuresKind = futureOptions.FuturesKind

	if len(options.Markets) == 0 {
		instance.BinanceFutureWs.Option.Markets, _ = instance.BinanceFutureRest.FetchMarkets()
	}
	return instance
}