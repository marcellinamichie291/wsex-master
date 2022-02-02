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

	"github.com/shiguantian/wsex"
)

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
			case wsex.MsgKLine:
				klines, ok := msg.Data.([]wsex.KLine)
				if !ok {
					fmt.Printf("kline data error %v", msg)
				}
				fmt.Printf("kline:%+v\n", klines)
			case wsex.MsgMarkPrice:
				markPrice, ok := msg.Data.(wsex.MarkPrice)
				if !ok {
					fmt.Printf("markprice data error %v", msg)
				}
				fmt.Printf("kline:%+v\n", markPrice)
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
				positions, ok := msg.Data.(wsex.FuturePositonsUpdate)
				if !ok {
					fmt.Printf("order data error %v", msg)
				}
				fmt.Printf("order:%+v\n", positions)
			}
		}
	}
}

func TestZbFutureWs_SubscribeOrderBook(t *testing.T) {
	//if _, err := e.SubscribeOrderBook(symbol, 20, 0, true, msgChan); err == nil {
	//	go handleMsg(msgChan)
	//}
	if _, err := zbFuture.SubscribeOrderBook(symbol, 50, 0, true, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}

func TestZbFutureWs_SubscribeTicker(t *testing.T) {
	if _, err := zbFuture.SubscribeTicker(symbol, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}

func TestZbFutureWs_SubscribeTrades(t *testing.T) {
	if _, err := zbFuture.SubscribeTrades(symbol, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}

func TestZbFutureWs_SubscribeKLine(t *testing.T) {
	if _, err := zbFuture.SubscribeKLine(symbol, wsex.KLine1Minute, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}

func TestZbFutureWs_SubscribeMarkPrice(t *testing.T) {
	if _, err := zbFuture.SubscribeMarkPrice(symbol, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}

func TestZbFutureWs_SubscribeBalance(t *testing.T) {
	if _, err := zbFuture.SubscribeBalance(symbol, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}
func TestZbFutureWs_SubscribeOrder(t *testing.T) {
	if _, err := zbFuture.SubscribeOrder(symbol, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}

func TestZbFutureWs_SubscribePositions(t *testing.T) {
	if _, err := zbFuture.SubscribePositions(symbol, msgChan); err == nil {
		handleFutureMsg(msgChan)
	}
}
