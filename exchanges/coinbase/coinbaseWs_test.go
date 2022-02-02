package coinbase

import (
	"fmt"
	"testing"
	"time"

	"github.com/shiguantian/wsex"
)

var (
	symbol  = "BTC/USD"
	symbol1 = "FIL/USDT"
	e       = New(wsex.Options{AccessKey: "4f21c4e5-63e88575-b1rkuf4drg-c846b", SecretKey: "3e1ad08b-ff5a4171-d021a0fb-9943a", AutoReconnect: true})
	msgChan = make(wsex.MessageChan)
)

func handleMsg(msgChan <-chan wsex.Message) {
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
			}
		}
	}
}

func TestCoinBseWs_SubscribeTicker(t *testing.T) {
	if _, err := e.SubscribeTicker(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestCoinBseWs_SubscribeOrderBook(t *testing.T) {
	if _, err := e.SubscribeOrderBook(symbol, 0, 0, false, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestCoinBseWs_SubscribeTrade(t *testing.T) {
	if _, err := e.SubscribeTrades(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestCoinBseWs_SubscribeKline(t *testing.T) {
	if _, err := e.SubscribeKLine(symbol, wsex.KLine1Minute, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestCoinBseWs_UnSubscribe(t *testing.T) {
	topic, err := e.SubscribeOrderBook(symbol, 0, 0, false, msgChan)
	if err == nil {
		go handleMsg(msgChan)
	}
	time.Sleep(time.Second * 10)
	e.UnSubscribe(topic, msgChan)
	time.Sleep(time.Second * 10)
}
