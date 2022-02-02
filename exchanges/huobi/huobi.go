/*
@Time : 2021/5/28 1:45 下午
@Author : zhaoqinghai
@File : huobi
@Software: GoLand
*/
package huobi

import "github.com/shiguantian/wsex"

type Huobi struct {
	HuobiRest
	HuobiWs
}

func New(options wsex.Options) *Huobi {
	instance := &Huobi{}
	instance.HuobiRest.Init(options)
	instance.HuobiWs.Init(options)

	if len(options.Markets) == 0 {
		instance.HuobiWs.Option.Markets, _ = instance.FetchMarkets()
	}
	return instance
}
