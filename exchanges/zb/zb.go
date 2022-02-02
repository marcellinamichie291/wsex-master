/*
@Time : 2021/5/8 1:45 下午
@Author : shiguantian
@File : zb
@Software: GoLand
*/
package zb

import "github.com/shiguantian/wsex"

type Zb struct {
	ZbRest
	ZbWs
}

func New(options wsex.Options) *Zb {
	instance := &Zb{}
	instance.ZbRest.Init(options)
	instance.ZbWs.Init(options)

	if len(options.Markets) == 0 {
		instance.ZbWs.Option.Markets, _ = instance.FetchMarkets()
	}
	return instance
}
