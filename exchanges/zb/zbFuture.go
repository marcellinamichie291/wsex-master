/*
@Time : 2021/5/8 1:45 下午
@Author : shiguantian
@File : zb
@Software: GoLand
*/
package zb

import "github.com/shiguantian/wsex"

type ZbFuture struct {
	ZbFutureRest
	ZbFutureWs
}

func NewFuture(options wsex.Options, futureOptions wsex.FutureOptions) *ZbFuture {
	instance := &ZbFuture{}
	instance.ZbFutureRest.Init(options)
	instance.ZbFutureRest.accountType = futureOptions.FutureAccountType
	instance.ZbFutureRest.contractType = futureOptions.ContractType
	instance.ZbFutureRest.futuresKind = futureOptions.FuturesKind
	instance.ZbFutureWs.Init(options)
	instance.ZbFutureWs.accountType = futureOptions.FutureAccountType
	instance.ZbFutureWs.contractType = futureOptions.ContractType
	instance.ZbFutureWs.futuresKind = futureOptions.FuturesKind
	if len(options.Markets) == 0 {
		instance.ZbFutureWs.Option.Markets, _ = instance.FetchMarkets()
	}
	return instance
}
