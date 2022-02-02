# wsex
A Golang library integrated websocket API of cryptocurrency exchanges

## Supported Cryptocurrency Exchange
name|ver|doc
---|---:|---:
Binance|v1.0|[API](https://binance-docs.github.io/apidocs/#change-log)
Okex|v3.0|[API](https://www.okex.com/docs/en/)
Zb|v1.0|[API](https://www.zb.today/en/api)


## examples
```
var (
	symbol    = "BTC/USDT"
	symbol1   = "ETH/USDT"
	e         = wsex.New(exchanges.Options{AccessKey: "xxxx", SecretKey: "xxxx", PassPhrase: "", AutoReconnect: true})
)

func handleMsg(msgChan <-chan exchanges.Message) {
	for {
		select {
		case msg := <-msgChan:
			switch msg.Type {
			case exchanges.MsgReConnected:
				fmt.Println("reconnected..., restart subscribe")
			case exchanges.MsgDisConnected:
				fmt.Println("disconnected, stop use old data, waiting reconnect....")
			case exchanges.MsgClosed:
				fmt.Println("websocket closed, stop all")
			case exchanges.MsgError:
				fmt.Printf("error happend: %v\n", msg.Data)
				if err, ok := msg.Data.(exchanges.ExError); ok {
					if err.Code == exchanges.ErrInvalidDepth {
						// symbol := err.Data["symbol"]
						//depth data invalid, Do some cleanup work, wait for the latest data or resubscribe
					}
				}
			case exchanges.MsgOrderBook:
				orderbook, ok := msg.Data.(exchanges.OrderBook)
				if !ok {
					fmt.Printf("order book data error %v", msg)
				}
				fmt.Printf("order book:%+v\n", orderbook)
			case exchanges.MsgTicker:
				ticker, ok := msg.Data.(exchanges.Ticker)
				if !ok {
					fmt.Printf("ticker data error %v", msg)
				}
				fmt.Printf("ticker:%+v\n", ticker)
			case exchanges.MsgTrade:
				trade, ok := msg.Data.(exchanges.Trade)
				if !ok {
					fmt.Printf("trade data error %v", msg)
				}
				fmt.Printf("trade:%+v\n", trade)
			case exchanges.MsgKLine:
				klines, ok := msg.Data.(exchanges.KLine)
				if !ok {
					fmt.Printf("kline data error %v", msg)
				}
				fmt.Printf("kline:%+v\n", klines)
			case exchanges.MsgBalance:
				balances, ok := msg.Data.(exchanges.BalanceUpdate)
				if !ok {
					fmt.Printf("balance data error %v", msg)
				}
				fmt.Printf("balance:%+v\n", balances)
			case exchanges.MsgOrder:
				order, ok := msg.Data.(exchanges.Order)
				if !ok {
					fmt.Printf("order data error %v", msg)
				}
				fmt.Printf("order:%+v\n", order)
			}
		}
	}
}

func TestBinance_SubscribeOrderBook(t *testing.T) {
	go func() {
		if _, msgChan, err := e.SubscribeOrderBook(symbol, 0, 0, true); err == nil {
			handleMsg(msgChan)
		}
	}()
	if _, msgChan, err := e.SubscribeOrderBook(symbol1, 0, 0, true); err == nil {
		handleMsg(msgChan)
	}
}

func TestBinance_SubscribeTicker(t *testing.T) {
	if _, msgChan, err := e.SubscribeTicker(symbol); err == nil {
		handleMsg(msgChan)
	}
}

func TestBinance_SubscribeTrades(t *testing.T) {
	if _, msgChan, err := e.SubscribeTrades(symbol); err == nil {
		handleMsg(msgChan)
	}
}

func TestBinance_SubscribeKLine(t *testing.T) {
	if _, msgChan, err := e.SubscribeKLine(symbol, exchanges.KLine1Minute); err == nil {
		handleMsg(msgChan)
	}
}

func TestBinance_SubscribeBalance(t *testing.T) {
	if _, msgChan, err := e.SubscribeBalance(symbol); err == nil {
		handleMsg(msgChan)
	}
}
func TestBinance_SubscribeOrder(t *testing.T) {
	if _, msgChan, err := e.SubscribeOrder(symbol); err == nil {
		handleMsg(msgChan)
	}
}
```
