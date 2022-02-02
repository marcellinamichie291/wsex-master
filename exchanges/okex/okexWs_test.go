/*
@Time : 2021/4/26 3:09 下午
@Author : shiguantian
@File : okex_test
@Software: GoLand
*/
package okex

import (
	"fmt"
	"testing"

	"github.com/shiguantian/wsex"
)

var (
	symbol  = "BTC/USDT"
	e       = New(wsex.Options{AccessKey: "", SecretKey: "", PassPhrase: "", AutoReconnect: true})
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
				ticker, ok := msg.Data.([]wsex.Ticker)
				if !ok {
					fmt.Printf("ticker data error %v", msg)
				}
				fmt.Printf("ticker:%+v\n", ticker)
			case wsex.MsgTrade:
				trades, ok := msg.Data.([]wsex.Trade)
				if !ok {
					fmt.Printf("trade data error %v", msg)
				}
				fmt.Printf("trade:%+v\n", trades)
			case wsex.MsgKLine:
				klines, ok := msg.Data.([]wsex.KLine)
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

func TestOkexWs_SubscribeOrderBook(t *testing.T) {
	if _, err := e.SubscribeOrderBook(symbol, 0, 0, true, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestOkexWs_SubscribeTicker(t *testing.T) {
	if _, err := e.SubscribeTicker(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestOkexWs_SubscribeTrades(t *testing.T) {
	if _, err := e.SubscribeTrades(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestOkexWs_SubscribeKLine(t *testing.T) {
	if _, err := e.SubscribeKLine(symbol, wsex.KLine1Minute, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestOkexWs_SubscribeBalance(t *testing.T) {
	if _, err := e.SubscribeBalance(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}
func TestOkexWs_SubscribeOrder(t *testing.T) {
	if _, err := e.SubscribeOrder(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}
