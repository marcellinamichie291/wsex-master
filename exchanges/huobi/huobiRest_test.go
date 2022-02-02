package huobi

import (
	"testing"

	"github.com/shiguantian/wsex"
)

var huobi = New(wsex.Options{AccessKey: "4f21c4e5-63e88575-b1rkuf4drg-c846b", SecretKey: "3e1ad08b-ff5a4171-d021a0fb-9943a", ProxyUrl: "http://127.0.0.1:4780"})

func TestHuobiRest_FetchTicker(t *testing.T) {
	ticker, err := huobi.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestHuobiRest_FetchOrderBook(t *testing.T) {
	res, err := huobi.FetchOrderBook("ETH/USDT", 0)
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_FetchAllTicker(t *testing.T) {
	res, err := huobi.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_FetchTrade(t *testing.T) {
	res, err := huobi.FetchTrade("ETH/USDT")
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_FetchKLine(t *testing.T) {
	res, err := huobi.FetchKLine("ETH/USDT", wsex.KLine1Day)
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_FetchOrder(t *testing.T) {
	res, err := huobi.FetchOrder("ETH/USDT", "320896583379611")
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_FetchBalance(t *testing.T) {
	res, err := huobi.FetchBalance()
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_CreateOrder(t *testing.T) {
	res, err := huobi.CreateOrder("ETH/USDT", 2000, 0.004, wsex.Buy, wsex.LIMIT, wsex.PostOnly, false)
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestHuobiRest_CancelOrder(t *testing.T) {
	err := huobi.CancelOrder("ETH/USDT", "320896557910586")
	if err != nil {
		t.Error(err)
	}
}

func TestHuobiRest_CancelAllOrders(t *testing.T) {
	err := huobi.CancelAllOrders("ETH/USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestHuobiRest_FetchAllOrders(t *testing.T) {
	orders, err := huobi.FetchOpenOrders("ETH/USDT", 0, 1000)
	if err != nil {
		t.Error(err)
	}
	t.Log(orders)
}
