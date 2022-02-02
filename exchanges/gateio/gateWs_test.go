package gateio

import (
	"fmt"
	"testing"

	"github.com/shiguantian/wsex"
)

var symbols = "EOS/USDT"

var e = New(wsex.Options{
	ExchangeName: "gate",
	SecretKey:    "",
	AccessKey:    "",
	ProxyUrl:     "http://127.0.0.1:4780",
})

var msgChan = make(wsex.MessageChan)

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
						// symbol := err.Data["symbol"]
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
				trade, ok := msg.Data.(wsex.Trade)
				if !ok {
					fmt.Printf("trade data error %v", msg)
				}
				fmt.Printf("trade:%+v\n", trade)
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

func TestGateWs_SubscribeOrderBook(t *testing.T) {
	_, err := e.SubscribeOrderBook(symbols, 200, 1000, false, msgChan)
	if err == nil {
		handleMsg(msgChan)
	}
}

func TestGateWs_SubscribeTicker(t *testing.T) {
	_, err := e.SubscribeTicker(symbols, msgChan)
	if err == nil {
		handleMsg(msgChan)
	}
}

func TestGateWs_SubscribeTrades(t *testing.T) {
	_, err := e.SubscribeTrades(symbols, msgChan)
	if err == nil {
		handleMsg(msgChan)
	}
}



func TestGateWs_SubscribeKLine(t *testing.T) {
	_, err := e.SubscribeKLine(symbols,wsex.KLine1Minute ,msgChan)
	if err == nil {
		handleMsg(msgChan)
	}
}

func TestGateWs_SubscribeBalance(t *testing.T) {
	_, err := e.SubscribeBalance(symbol, msgChan)
	if err == nil {
		handleMsg(msgChan)
	}
}

func TestGateWs_SubscribeOrder(t *testing.T) {
	_, err := e.SubscribeOrder(symbol, msgChan)
	if err == nil {
		handleMsg(msgChan)
	}
}

func TestGateWs_getSnapshotOrderBook(t *testing.T){
	var sym =make(SymbolOrderBook)
 	err:=e.getSnapshotOrderBook("2",wsex.Market{
 		SymbolID: "EOS_USDT",
 		Symbol: symbol,
	},&sym)
 	if err!=nil{
 		fmt.Println(err)
	}
	fmt.Println((*e.orderBooks["2"])[symbol].OrderBook)

}
