/*
@Time : 2021/4/20 4:31 下午
@Author : shiguantian
@File : zb_test
@Software: GoLand
*/
package zb

import (
	"fmt"
	"testing"
	"time"

	"github.com/shiguantian/wsex"
)

var (
	symbol  = "ETH/USDT"
	symbol1 = "FIL/USDT"
	e       = New(wsex.Options{AccessKey: "aa36b439-b91d-408b-90df-13e45d3ebab4", SecretKey: "43c6ea05-5e82-42d0-9612-3ac5b8947f1f", AutoReconnect: true})
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

func TestZbWs_SubscribeOrderBook(t *testing.T) {
	//if _, err := e.SubscribeOrderBook(symbol, 20, 0, true, msgChan); err == nil {
	//	go handleMsg(msgChan)
	//}
	if _, err := e.SubscribeOrderBook(symbol1, 20, 0, true, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestZbWs_SubscribeTicker(t *testing.T) {
	if _, err := e.SubscribeTicker(symbol, msgChan); err == nil {
		go handleMsg(msgChan)
	}
	if _, err := e.SubscribeTicker(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestZbWs_SubscribeTrades(t *testing.T) {
	if _, err := e.SubscribeTrades(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestZbWs_SubscribeKLine(t *testing.T) {
	if _, err := e.SubscribeKLine(symbol, wsex.KLine1Minute, msgChan); err == nil {
		handleMsg(msgChan)
	}
}

func TestZbWs_SubscribeBalance(t *testing.T) {
	if _, err := e.SubscribeBalance(symbol, msgChan); err == nil {
		handleMsg(msgChan)
	}
}
func TestZbWs_SubscribeOrder(t *testing.T) {
	if _, err := e.SubscribeOrder("FIL/USDT", msgChan); err == nil {
		go func() {
			for {
				time.Sleep(time.Millisecond * 20)
				order, err := e.CreateOrder("FIL/USDT", 10, 1, wsex.Buy, wsex.LIMIT, wsex.Normal, false)
				fmt.Printf("%v create order %v\n", time.Now(), order.ID)
				if err == nil {
					e.CancelOrder("FIL/USDT", order.ID)
					fmt.Printf("%v cancel order %v\n", time.Now(), order.ID)
				}
			}
		}()
		handleMsg(msgChan)
	}
}
