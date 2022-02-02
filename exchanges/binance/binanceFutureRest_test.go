package binance

import (
	"testing"

	"github.com/shiguantian/wsex"
)

var baFuture = NewFuture(wsex.Options{
	AccessKey: "X6zdvDI3cvSaqnQAbOBT1QF89lUEfy9fXcr859SI9R3AH34fvOuDoAAsXzt3Dt1h",
	SecretKey: "Em0XgHLEGzxvw24GWocIS8kTnxo7gVJXJxPJ6F25b8Mxor0CNU9dyudaBnaVKNqe",
	ProxyUrl:  "http://127.0.0.1:4780"}, wsex.FutureOptions{
	ContractType:      wsex.Swap,
	FutureAccountType: wsex.UsdtMargin,
})

func TestBinanceFutureRest_FetchMarkets(t *testing.T) {
	markets, err := baFuture.FetchMarkets()
	if err != nil {
		t.Error(err)
	}
	t.Log(markets["BTC/USDT"])
}

func TestBinanceFutureRest_FetchOrderBook(t *testing.T) {
	orderBook, err := baFuture.FetchOrderBook(symbol1, 5)
	if err != nil {
		t.Error(err)
	}
	t.Log(orderBook)
}

func TestBinanceFutureRest_FetchTicker(t *testing.T) {
	ticker, err := baFuture.FetchTicker(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(ticker)
}

func TestBinanceFutureRest_FetchAllTicker(t *testing.T) {
	tickers, err := baFuture.FetchAllTicker()
	if err != nil {
		t.Error(err)
	}
	t.Log(tickers)
}

func TestBinanceFutureRest_FetchTrade(t *testing.T) {
	trade, err := baFuture.FetchTrade(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(trade)
}

func TestBinanceFutureRest_FetchKLine(t *testing.T) {
	klines, err := baFuture.FetchKLine(symbol, wsex.KLine1Hour)
	if err != nil {
		t.Error(err)
	}
	t.Log(klines)
}

func TestBinanceFutureRest_CreateOrder(t *testing.T) {
	order, err := baFuture.CreateOrder(symbol, 28500, 0.025, wsex.OpenLong, wsex.LIMIT, wsex.Normal, false)
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestBinanceFutureRest_CancelOrder(t *testing.T) {
	err := baFuture.CancelOrder(symbol, "29666601144")
	if err != nil {
		t.Error(err)
	}
}

func TestBinanceFutureRest_CancelAllOrders(t *testing.T) {
	err := baFuture.CancelAllOrders(symbol)
	if err != nil {
		t.Error(err)
	}
}

func TestBinanceFutureRest_FetchOrder(t *testing.T) {
	order, err := baFuture.FetchOrder(symbol, "29676698858")
	if err != nil {
		t.Error(err)
	}
	t.Log(order)
}

func TestBinanceFutureRest_FetchOpenOrders(t *testing.T) {
	orders, err := baFuture.FetchOpenOrders(symbol, 1, 10)
	if err != nil {
		t.Error(err)
	}
	t.Log(orders)
}

func TestBinanceFutureRest_FetchBalance(t *testing.T) {
	Balances, err := baFuture.FetchBalance()
	if err != nil {
		t.Error(err)
	}
	t.Log(Balances)
}

func TestBinanceFutureRest_FetchAccountInfo(t *testing.T) {
	accountInfo, err := baFuture.FetchAccountInfo()
	if err != nil {
		t.Error(err)
	}
	t.Log(accountInfo)
}

func TestBinanceFutureRest_FetchPositions(t *testing.T) {
	positions, err := baFuture.FetchPositions("")
	if err != nil {
		t.Error(err)
	}
	t.Log(positions)
}

func TestBinanceFutureRest_FetchAllPositions(t *testing.T) {
	positions, err := baFuture.FetchAllPositions()
	if err != nil {
		t.Error(err)
	}
	t.Log(positions)
}

func TestBinanceFutureRest_FetchMarkPrice(t *testing.T) {
	markPrice, err := baFuture.FetchMarkPrice("LTC/USDT")
	if err != nil {
		t.Error(err)
	}
	t.Log(markPrice)
}

func TestBinanceFutureRest_FetchFundingRate(t *testing.T) {
	fundingrate, err := baFuture.FetchFundingRate(symbol)
	if err != nil {
		t.Error(err)
	}
	t.Log(fundingrate)
}

func TestBinanceFutureRest_Setting(t *testing.T) {
	err := baFuture.Setting(symbol, 5, wsex.CrossedMargin, wsex.TwoWay)
	if err != nil {
		t.Error(err)
	}
}
