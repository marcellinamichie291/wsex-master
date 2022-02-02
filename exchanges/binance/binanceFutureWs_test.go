package binance

import (
	"fmt"
	"testing"

	"github.com/shiguantian/wsex"
)

var BaFuture = NewFuture(wsex.Options{
	PassPhrase: "",
	ProxyUrl:   "http://127.0.0.1:4780"}, wsex.FutureOptions{
	FutureAccountType: wsex.UsdtMargin,
	ContractType:      wsex.Swap,
})
var fmsgChan = make(wsex.MessageChan)

func handleFutureMsg(msgChan <-chan wsex.Message) {
	for {
		select {
		case msg := <-msgChan:
			switch msg.Type {
			case wsex.MsgReConnected:
				fmt.Println("reconnected..., restart subscribe")
			case wsex.MsgDisConnected:
				fmt.Println("disconnected, stop use old data, waiting reconnect....")
			case wsex.MsgClosed:
				fmt.Println("websocket closed, stop all")
				break
			case wsex.MsgError:
				fmt.Printf("error happend: %v\n", msg.Data)
				if err, ok := msg.Data.(wsex.ExError); ok {
					if err.Code == wsex.ErrInvalidDepth {
						//depth data invalid, Do some cleanup work, wait for the latest data or resubscribe
					}
				}
			case wsex.MsgOrderBook:
				orderbook, ok := msg.Data.(wsex.OrderBook)
				if !ok {
					fmt.Printf("order book data error %v", msg)
				}
				fmt.Printf("order book:%+v\n", orderbook)
			case wsex.MsgTicker:
				ticker, ok := msg.Data.(wsex.Ticker)
				if !ok {
					fmt.Printf("ticker data error %v", msg)
				}
				fmt.Printf("ticker:%+v\n", ticker)
			case wsex.MsgTrade:
				trades, ok := msg.Data.(wsex.Trade)
				if !ok {
					fmt.Printf("trade data error %v", msg)
				}
				fmt.Printf("trade:%+v\n", trades)
			case wsex.MsgMarkPrice:
				markPrice, ok := msg.Data.(wsex.MarkPrice)
				if !ok {
					fmt.Printf("markprice data error %v", msg)
				}
				fmt.Printf("markPrice:%+v\n", markPrice)
			case wsex.MsgKLine:
				klines, ok := msg.Data.(wsex.KLine)
				if !ok {
					fmt.Printf("kline data error %v", msg)
				}
				fmt.Printf("kline:%+v\n", klines)
			case wsex.MsgBalance:
				balances, ok := msg.Data.(wsex.BalanceUpdate)
				if !ok {
					fmt.Printf("balance data error %v", msg)
				}
				fmt.Printf("balance:%+v\n", balances)
			case wsex.MsgOrder:
				order, ok := msg.Data.(wsex.Order)
				if !ok {
					fmt.Printf("order data error %v", msg)
				}
				fmt.Printf("order:%+v\n", order)
			case wsex.MsgPositions:
				position, ok := msg.Data.(wsex.FuturePositonsUpdate)
				if !ok {
					fmt.Printf("order data error %v", msg)
				}
				fmt.Printf("position:%+v\n", position)
			}
		}
	}
}

func TestBinanceFutureWs_SubscribeOrderBook(t *testing.T) {

	if _, err := BaFuture.SubscribeOrderBook(symbol, 5, 0, true, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}

func TestBinanceFutureWs_SubscribeTicker(t *testing.T) {
	if _, err := BaFuture.SubscribeTicker(symbol, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}

func TestBinanceFutureWs_SubscribeTrades(t *testing.T) {
	if _, err := BaFuture.SubscribeTrades(symbol, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}

func TestBinanceFutureWs_SubscribeKLine(t *testing.T) {
	if _, err := BaFuture.SubscribeKLine(symbol, wsex.KLine3Minute, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}

func TestBinanceFutureWs_SubscribeBalance(t *testing.T) {
	if _, err := BaFuture.SubscribeBalance(symbol, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}
func TestZbFutureWs_SubscribeOrder(t *testing.T) {
	if _, err := BaFuture.SubscribeOrder(symbol, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}

func TestBinanceFutureWs_SubscribePositions(t *testing.T) {
	if _, err := BaFuture.SubscribePositions(symbol, fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}

func TestBinanceFutureWs_SubscribeMarkPrice(t *testing.T) {
	if _, err := BaFuture.SubscribeMarkPrice("EOS/USDT", fmsgChan); err == nil {
		handleFutureMsg(fmsgChan)
	}
}
