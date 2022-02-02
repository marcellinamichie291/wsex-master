/*
@Time : 2021/5/10 1:00 下午
@Author : shiguantian
@File : binance
@Software: GoLand
*/
package binance

import "github.com/shiguantian/wsex"

type Binance struct {
	BinanceRest
	BinanceWs
}

func New(options wsex.Options) *Binance {
	instance := &Binance{}
	instance.BinanceRest.Init(options)
	instance.BinanceWs.Init(options)

	if len(options.Markets) == 0 {
		instance.BinanceWs.Option.Markets, _ = instance.FetchMarkets()
	}

	return instance
}
