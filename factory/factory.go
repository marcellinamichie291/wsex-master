/*
@Time : 2021/5/7 1:19 下午
@Author : shiguantian
@File : factory
@Software: GoLand
*/
package factory

import (
	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges/binance"
	"github.com/shiguantian/wsex/exchanges/gateio"
	"github.com/shiguantian/wsex/exchanges/huobi"
	"github.com/shiguantian/wsex/exchanges/okex"
	"github.com/shiguantian/wsex/exchanges/zb"
)

func NewExchange(t wsex.ExchangeType, option wsex.Options) wsex.IExchange {
	switch t {
	case wsex.Binance:
		return binance.New(option)
	case wsex.ZB:
		return zb.New(option)
	case wsex.Okex:
		return okex.New(option)
	case wsex.Huobi:
		return huobi.New(option)
	case wsex.GateIo:
		return gateio.New(option)
	}
	return nil
}

func NewFutureExchange(t wsex.ExchangeType, option wsex.Options, futureOptions wsex.FutureOptions) wsex.IFutureExchange {
	switch t {
	case wsex.Binance:
		return binance.NewFuture(option,futureOptions)
	case wsex.ZB:
		return zb.NewFuture(option, futureOptions)
	}
	return nil
}
