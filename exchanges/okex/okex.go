/*
@Time : 2021/5/9 2:33 下午
@Author : shiguantian
@File : okex
@Software: GoLand
*/
package okex

import (
	"github.com/shiguantian/wsex"
)

type Okex struct {
	OkexRest
	OkexWs
}

func New(options wsex.Options) *Okex {
	instance := &Okex{}
	instance.OkexRest.Init(options)
	instance.OkexWs.Init(options)

	if len(options.Markets) == 0 {
		instance.OkexWs.Option.Markets, _ = instance.FetchMarkets()
	}

	return instance
}
