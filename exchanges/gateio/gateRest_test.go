package gateio

import (
	"testing"

	"github.com/shiguantian/wsex"
)

var symbol = "EOS/USDT"

var rest = New(wsex.Options{
	ExchangeName: "gate",
	SecretKey:    "",
	AccessKey:    "",
})

func TestGateRest_FetchOrderBook(t *testing.T) {
	orderbook, err := rest.FetchOrderBook(symbol, 0)
	if err != nil {
		t.Error(err)
	}
	t.Log(orderbook)
}

func TestGateRest_FetchTicker(t *testing.T) {
	ticker, err := rest.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestGateRest_FetchAllTicker(t *testing.T) {
	tickers, err := rest.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	t.Log(tickers)
}

func TestGateRest_FetchTrade(t *testing.T) {
	trades, err := rest.FetchTrade(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(trades)
}

func TestGateRest_FetchKLine(t *testing.T) {
	Klines, err := rest.FetchKLine(symbol, wsex.KLine15Minute)
	if err != nil {
		t.Error(err)
	}
	t.Log(Klines)
}

func TestGateRest_FetchMarkets(t *testing.T) {
	markets, err := rest.FetchMarkets()
	if err != nil {
		t.Error(err)
	}
	t.Log(markets)
}

func TestGateRest_FetchBalance(t *testing.T) {
	balances, err := rest.FetchBalance()
	if err != nil {
		t.Error(err)
	}
	t.Log(balances)
}

func TestGateRest_CreateOrder(t *testing.T) {
	order, err := rest.CreateOrder(symbol, 3, 1.53, wsex.Buy, wsex.LIMIT, wsex.Normal, false)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestGateRest_CancelOrder(t *testing.T) {
	err := rest.CancelOrder(symbol, "65912650132")
	if err != nil {
		t.Error(err)
	}
}
func TestGateRest_CancelAllOrders(t *testing.T) {
	err := rest.CancelAllOrders(symbol)
	if err != nil {
		t.Error(err)
	}
}

func TestGateRest_FetchOrder(t *testing.T) {
	order, err := rest.FetchOrder(symbol, "65912650132")
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestGateRest_FetchOpenOrders(t *testing.T) {
	order, err := rest.FetchOpenOrders(symbol, 0, 0)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}
