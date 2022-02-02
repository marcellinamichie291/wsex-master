/*
@Time : 2021/5/8 4:37 下午
@Author : shiguantian
@File : zbRest_test
@Software: GoLand
*/
package zb

import (
	"fmt"
	"testing"
	"time"

	"github.com/shiguantian/wsex"
)

var zbFuture = NewFuture(wsex.Options{
	AccessKey: "81813557-38f3-472a-b33c-2f40b03dddb8",
	SecretKey: "5b6ba6aa-6888-4028-8009-b20d49813f48",
	ProxyUrl:  "http://127.0.0.1:4780",
	//RestHost:        "https://futures.zb.work",
	//RestPrivateHost: "https://futures.zb.work",
	//WsHost:          "wss://futures.zb.work/ws",
}, wsex.FutureOptions{
	FutureAccountType: wsex.UsdtMargin,
	ContractType:      wsex.Swap,
})

func TestZbFutureRest_FetchMarkets(t *testing.T) {
	market, err := zbFuture.FetchMarkets()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", market)
}

func TestZbFutureRest_FetchOrderBook(t *testing.T) {
	orderBook, err := zbFuture.FetchOrderBook(symbol, 50)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", orderBook)
}

func TestZbFutureRest_FetchTicker(t *testing.T) {
	ticker, err := zbFuture.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", ticker)
}

func TestZbFutureRest_FetchAllTicker(t *testing.T) {
	tickers, err := zbFuture.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", tickers)
}

func TestZbFutureRest_FetchTrade(t *testing.T) {
	trade, err := zbFuture.FetchTrade(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(trade)
}

func TestZbFutureRest_FetchKLine(t *testing.T) {
	klines, err := zbFuture.FetchKLine(symbol, wsex.KLine1Hour)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", klines)
}

func TestZbFutureRest_FetchMarkPrice(t *testing.T) {
	markPrice, err := zbFuture.FetchMarkPrice(symbol)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", markPrice)
}

func TestZbFutureRest_FetchFundingRate(t *testing.T) {
	markPrice, err := zbFuture.FetchFundingRate(symbol)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", markPrice)
}

func TestZbFutureRest_FetchBalance(t *testing.T) {
	balances, err := zbFuture.FetchBalance()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", balances)
}

func TestZbFutureRest_FetchAccountInfo(t *testing.T) {
	accountInfo, err := zbFuture.FetchAccountInfo()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", accountInfo)
}

func TestZbFutureRest_FetchPositions(t *testing.T) {
	positions, err := zbFuture.FetchPositions("")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%v", positions)
}

func TestZbFutureRest_Setting(t *testing.T) {
	err := zbFuture.Setting(symbol, 1, wsex.FixedMargin, wsex.TwoWay)
	if err != nil {
		t.Error(err)
	}
}

func TestZbFutureRest_CreateOrder(t *testing.T) {
	order, err := zbFuture.CreateOrder(symbol, 45700, 1, wsex.OpenLong, wsex.LIMIT, wsex.PostOnly, false)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestZbFutureRest_FetchOrder(t *testing.T) {
	order, err := zbFuture.FetchOrder(symbol, "6834129972714741390")
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestZbFutureRest_FetchOpenOrders(t *testing.T) {
	orders, err := zbFuture.FetchOpenOrders(symbol, 1, 10)
	if err != nil {
		t.Error(err)
	}
	t.Log(orders)
}

func TestZbFutureRest_CancelOrder(t *testing.T) {
	err := zbFuture.CancelOrder(symbol, "6834129972714741760")
	if err != nil {
		t.Error(err)
	}
}

func TestZbFutureRest_CancelAllOrders(t *testing.T) {
	err := zbFuture.CancelAllOrders(symbol)
	if err != nil {
		t.Error(err)
	}
}

func TestZbFutureRest_Pressure(t *testing.T) {
	for {
		time.Sleep(time.Second)
		order, err := zbFuture.CreateOrder(symbol, 40000, 0.1, wsex.OpenLong, wsex.LIMIT, wsex.PostOnly, false)
		if err != nil {
			t.Error(err)
		}
		println("place ok")
		time.Sleep(time.Second)
		err = zbFuture.CancelOrder(symbol, order.ID)
		if err != nil {
			t.Error(err)
		}
		println("cancel ok")
		time.Sleep(time.Second)
	}
}
