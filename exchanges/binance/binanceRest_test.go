/*
@Time : 2021/5/10 2:39 下午
@Author : shiguantian
@File : binanceRest_test
@Software: GoLand
*/
package binance

import (
	"testing"

	"github.com/shiguantian/wsex"
)

var rest = New(wsex.Options{AccessKey: "R2UqsV4awpcG4wQX83GRYYCSuC4NXspKQPLiIcujTtWLdvIzZcxf61Hi3lhHxy76", SecretKey: "h0glsQ9XAYMB9E09HR2Xtyxyt72STyZZ2Gm8KiKnQnuIMW8DIwujfIGp4OJjiXvJ", PassPhrase: "", ProxyUrl: "http://127.0.0.1:4780"})
var orderID string

func TestBinanceRest_FetchMarkets(t *testing.T) {
	markets, err := rest.FetchMarkets()
	if err != nil {
		t.Error(err)
	}
	t.Log(markets)
}

func TestBinanceRest_FetchOrderBook(t *testing.T) {
	orderBook, err := rest.FetchOrderBook(symbol, 50)
	if err != nil {
		t.Error(err)
	}
	t.Log(orderBook)
}

func TestBinanceRest_FetchTicker(t *testing.T) {
	ticker, err := rest.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestBinanceRest_FetchAllTicker(t *testing.T) {
	tickers, err := rest.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	t.Log(tickers)
}

func TestBinanceRest_FetchTrade(t *testing.T) {
	trade, err := rest.FetchTrade(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(trade)
}

func TestBinanceRest_FetchKLine(t *testing.T) {
	klines, err := rest.FetchKLine(symbol, wsex.KLine1Day)
	if err != nil {
		t.Error(err)
	}
	t.Log(klines)
}

func TestBinanceRest_FetchBalance(t *testing.T) {
	balances, err := rest.FetchBalance()
	if err != nil {
		t.Error(err)
	}
	t.Log(balances)
}

func TestBinanceRest_CreateOrder(t *testing.T) {
	order, err := rest.CreateOrder(symbol, 30000, 0.001, wsex.Buy, wsex.LIMIT, wsex.Normal, false)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestBinanceRest_FetchOrder(t *testing.T) {
	order, err := rest.FetchOrder(symbol, "6938997229316096")
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestBinanceRest_FetchOpenOrders(t *testing.T) {
	orders, err := rest.FetchOpenOrders(symbol, 1, 10)
	if err != nil {
		t.Error(err)
	}
	t.Log(orders)
}

func TestBinanceRest_CancelOrder(t *testing.T) {
	//order, err := rest.CreateOrder(symbol, 10000, 0.001, wsex.Buy, wsex.LIMIT, wsex.Normal, false)
	err := rest.CancelOrder(symbol, "6938997229316096")
	if err != nil {
		t.Error(err)
	}
}
