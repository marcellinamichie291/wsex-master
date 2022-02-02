package coinbase

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/exchanges/websocket"
)

type Stream map[string]string

type SubTopic struct {
	Topic        string
	Symbol       string
	MessageType  wsex.MessageType
	LastUpdateID float64
}

type CoinBaseWs struct {
	exchanges.BaseExchange
	orderBooks   map[string]*CoinBaseOrderBook
	subTopicInfo map[string]SubTopic
	errors       map[int]wsex.ExError
	loginLock    sync.Mutex
	loginChan    chan struct{}
	isLogin      bool
}

func (e *CoinBaseWs) Init(option wsex.Options) {
	e.BaseExchange.Init()
	e.Option = option
	e.orderBooks = make(map[string]*CoinBaseOrderBook)
	e.subTopicInfo = make(map[string]SubTopic)
	e.errors = map[int]wsex.ExError{}
	e.loginLock = sync.Mutex{}
	e.loginChan = make(chan struct{})
	e.isLogin = false
	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://ws-feed.pro.coinbase.com"
	}
}

func (e *CoinBaseWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	return e.subscribe(e.Option.WsHost, market.SymbolID, symbol, wsex.MsgOrderBook, false, sub)
}

func (e *CoinBaseWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	return e.subscribe(e.Option.WsHost, market.SymbolID, symbol, wsex.MsgTicker, false, sub)
}

func (e *CoinBaseWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	return e.subscribe(e.Option.WsHost, market.SymbolID, symbol, wsex.MsgTrade, false, sub)
}

func (e *CoinBaseWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *CoinBaseWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *CoinBaseWs) UnSubscribe(event string, sub wsex.MessageChan) error {
	conn, err := e.ConnectionMgr.GetConnection(e.Option.WsHost, nil)
	if err != nil {
		return err
	}
	tuple := strings.Split(event, "#")
	data := make(map[string]interface{}, 3)
	if len(tuple) == 2 {
		data = map[string]interface{}{
			"product_ids": []string{tuple[0]},
			"type":        "subscribe",
			"channels":    []string{tuple[1]},
		}
	}
	if err := conn.SendJsonMessage(data); err != nil {
		return err
	}
	conn.UnSubscribe(sub)
	return nil
}

func (e *CoinBaseWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *CoinBaseWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *CoinBaseWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("CoinBase"),
		websocket.SetWsUrl(url),
		websocket.SetIsAutoReconnect(e.Option.AutoReconnect),
		websocket.SetEnableCompression(false),
		websocket.SetReadDeadLineTime(time.Minute),
		websocket.SetMessageHandler(e.messageHandler),
		websocket.SetErrorHandler(e.errorHandler),
		websocket.SetCloseHandler(e.closeHandler),
		websocket.SetReConnectedHandler(e.reConnectedHandler),
		websocket.SetDisConnectedHandler(e.disConnectedHandler),
	)
	return conn, err
}

func (e *CoinBaseWs) subscribe(url, topic, symbol string, t wsex.MessageType, needLogin bool, sub wsex.MessageChan) (string, error) {
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	if needLogin {

	} else {
		data = map[string]interface{}{
			"product_ids": []string{topic},
			"type":        "subscribe",
		}
		switch t {
		case wsex.MsgTicker:
			data["channels"] = []string{"ticker"}
			topic += "#ticker"
		case wsex.MsgTrade:
			data["channels"] = []string{"matches"}
			topic += "#matches"
		case wsex.MsgOrderBook:
			data["channels"] = []string{"level2"}
			topic += "#level2"
		}
	}

	if err := conn.SendJsonMessage(data); err != nil {
		return "", err
	}
	conn.Subscribe(sub)

	return topic, nil
}

func (e *CoinBaseWs) messageHandler(url string, message []byte) {
	res := Response{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] messageHandler unmarshal error:%v", err))
		return
	}
	switch res.Type {
	case "ticker":
		e.handleTicker(url, message)
	case "match":
		e.handleTrade(url, message)
	case "snapshot", "l2update":
		e.handleDepth(url, message, res.Type)
	}
}

func (e *CoinBaseWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
}

func (e *CoinBaseWs) disConnectedHandler(url string, err error) {
	// clear cache data, Prevent getting dirty data
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		e.isLogin = false
		delete(e.orderBooks, url)
	})
}

func (e *CoinBaseWs) closeHandler(url string) {
	// clear cache data and the connection
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
	})
}

func (e *CoinBaseWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *CoinBaseWs) handleTicker(url string, message []byte) {
	var data WsTickerRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleTicker - find market by id error:%v", err))
		return
	}
	ticker := data.parseTicker(market)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: ticker})
}

func (e *CoinBaseWs) handleDepth(url string, message []byte, depthType string) {
	if depthType == "snapshot" {
		var data OrderBookRes
		if err := json.Unmarshal(message, &data); err != nil {
			e.errorHandler(url, fmt.Errorf("[huobiWs] handleDepth - message Unmarshal to ticker error:%v", err))
			return
		}
		market, err := e.GetMarketByID(data.Symbol)
		if err != nil {
			e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleDepth - find market by id error:%v", err))
			return
		}
		symbolOrderBook, ok := e.orderBooks[market.Symbol]
		if !ok {
			symbolOrderBook = &CoinBaseOrderBook{}
			e.orderBooks[market.Symbol] = symbolOrderBook
		}
		var asks wsex.Depth = wsex.Depth{}
		for _, ask := range data.Asks {
			depthItem, err := ask.ParseRawDepthItem()
			if err != nil {
				e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleDepth - parse depth item error:%v", err))
			}
			asks = append(asks, depthItem)
		}
		var bids wsex.Depth = wsex.Depth{}
		for _, bid := range data.Bids {
			depthItem, err := bid.ParseRawDepthItem()
			if err != nil {
				e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleDepth - parse depth item error:%v", err))
			}
			bids = append(bids, depthItem)
		}
		sort.Sort(asks)
		sort.Sort(sort.Reverse(bids))
		symbolOrderBook.Asks = asks
		symbolOrderBook.Bids = bids
		symbolOrderBook.Symbol = market.Symbol
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: symbolOrderBook.OrderBook})
	} else {
		var data WsOrderBookUpdateRes
		if err := json.Unmarshal(message, &data); err != nil {
			e.errorHandler(url, fmt.Errorf("[huobiWs] handleDepth - message Unmarshal to ticker error:%v", err))
			return
		}
		market, err := e.GetMarketByID(data.Symbol)
		if err != nil {
			e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleDepth - find market by id error:%v", err))
			return
		}
		symbolOrderBook, ok := e.orderBooks[market.Symbol]
		if !ok {
			e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleDepth - find cache orderbook err:%v", err))
			return
		}
		var changeDepth OrderBookRes = OrderBookRes{
			Asks: wsex.RawDepth{},
			Bids: wsex.RawDepth{},
		}
		for _, change := range data.Changes {
			if change[0] == "sell" {
				changeDepth.Asks = append(changeDepth.Asks, wsex.RawDepthItem{
					change[1], change[2],
				})
			} else {
				changeDepth.Bids = append(changeDepth.Bids, wsex.RawDepthItem{
					change[1], change[2],
				})
			}
		}
		symbolOrderBook.update(changeDepth)

		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: symbolOrderBook.OrderBook})
	}
}

func (e *CoinBaseWs) handleTrade(url string, message []byte) {
	var data WsTradeRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[coinBaseWs] handleTicker - find market by id error:%v", err))
		return
	}
	trade := data.parseTrade(market)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: trade})
}
