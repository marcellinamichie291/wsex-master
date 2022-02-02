/*
@Time : 2021/5/8 4:37 下午
@Author : shiguantian
@File : zbRest_test
@Software: GoLand
*/
package zb

import (
	"testing"

	"github.com/shiguantian/wsex"
)

var zb = New(wsex.Options{
	AccessKey:       "aa36b439-b91d-408b-90df-13e45d3ebab4",
	SecretKey:       "43c6ea05-5e82-42d0-9612-3ac5b8947f1f",
	RestHost:        "https://api.zb.work",
	RestPrivateHost: "https://trade.zb.work",
})
var orderID string

func TestZbRest_FetchMarkets(t *testing.T) {
	if _, err := zb.FetchMarkets(); err != nil {
		t.Error(err)
	}
}

func TestZbRest_FetchOrderBook(t *testing.T) {
	orderBook, err := zb.FetchOrderBook(symbol, 50)
	if err != nil {
		t.Error(err)
	}
	t.Log(orderBook)
}

func TestZbRest_FetchTicker(t *testing.T) {
	ticker, err := zb.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestZbRest_FetchAllTicker(t *testing.T) {
	tickers, err := zb.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	t.Log(tickers)
}

func TestZbRest_FetchTrade(t *testing.T) {
	trade, err := zb.FetchTrade(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(trade)
}

func TestZbRest_FetchKLine(t *testing.T) {
	klines, err := zb.FetchKLine(symbol, wsex.KLine1Day)
	if err != nil {
		t.Error(err)
	}
	t.Log(klines)
}

func TestZbRest_FetchBalance(t *testing.T) {
	balances, err := zb.FetchBalance()
	if err != nil {
		t.Error(err)
	}
	t.Log(balances)
}

func TestZbRest_CreateOrder(t *testing.T) {
	order, err := zb.CreateOrder(symbol, 3000, 0.001, wsex.Buy, wsex.LIMIT, wsex.PostOnly, false)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestZbRest_FetchOrder(t *testing.T) {
	order, err := zb.FetchOrder(symbol, "202105093079666159")
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestZbRest_FetchOpenOrders(t *testing.T) {
	orders, err := zb.FetchOpenOrders(symbol, 1, 10)
	if err != nil {
		t.Error(err)
	}
	t.Log(orders)
}

func TestZbRest_CancelOrder(t *testing.T) {
	order, err := zb.CreateOrder(symbol, 10000, 0.001, wsex.Buy, wsex.LIMIT, wsex.Normal, false)
	err = zb.CancelOrder(symbol, order.ID)
	if err != nil {
		t.Error(err)
	}
}

func TestZbRest_CancelAllOrders(t *testing.T) {
	err := zb.CancelAllOrders(symbol)
	if err != nil {
		t.Error(err)
	}
}
