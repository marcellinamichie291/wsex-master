/* Time : 2021/5/28 1:45 下午
@Author : zhaoqinghai
@File : coinbase
@Software: GoLand
*/
package coinbase

import "github.com/shiguantian/wsex"

type CoinBase struct {
	CoinBaseRest
	CoinBaseWs
}

func New(options wsex.Options) *CoinBase {
	instance := &CoinBase{}
	instance.CoinBaseRest.Init(options)
	instance.CoinBaseWs.Init(options)

	if len(options.Markets) == 0 {
		instance.CoinBaseWs.Option.Markets, _ = instance.FetchMarkets()
	}
	return instance
}
