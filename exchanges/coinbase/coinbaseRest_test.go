package coinbase

import (
	"testing"

	"github.com/shiguantian/wsex"
)

var coinbase = New(wsex.Options{AccessKey: "4f21c4e5-63e88575-b1rkuf4drg-c846b", SecretKey: "3e1ad08b-ff5a4171-d021a0fb-9943a", ProxyUrl: "http://127.0.0.1:4780"})

func TestCoinBaseRest_FetchMarket(t *testing.T) {
	ticker, err := coinbase.FetchMarkets()
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestCoinBaseRest_FetchTicker(t *testing.T) {
	ticker, err := coinbase.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestCoinBaseRest_FetchOrderBook(t *testing.T) {
	order, err := coinbase.FetchOrderBook("ETH/USDT", 2)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestCoinBaseRest_FetchAllTicker(t *testing.T) {
	order, err := coinbase.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestCoinBaseRest_FetchTrade(t *testing.T) {
	order, err := coinbase.FetchTrade("ETH/USDT")
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestCoinBaseRest_FetchKLine(t *testing.T) {
	order, err := coinbase.FetchKLine("ETH/USDT", wsex.KLine15Minute)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}
