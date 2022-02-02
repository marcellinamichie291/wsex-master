package gateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"

	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/exchanges/websocket"
	"github.com/shiguantian/wsex/utils"
)

type Stream struct {
	Time    int64             `json:"time"`
	ID      int64             `json:"id,omitempty"`
	Channel string            `json:"channel"`
	Event   string            `json:"event,omitempty"`
	Payload []string          `json:"payload,omitempty"`
	Auth    map[string]string `json:"auth,omitempty"`
}

func SubscribeStream(channel string, payload []string) Stream {
	return Stream{
		Time:    time.Now().Unix(),
		Channel: channel,
		Event:   "subscribe",
		Payload: payload,
	}
}
func UnSubscribeStream(channel string, payload []string) Stream {
	return Stream{
		Time:    time.Now().Unix(),
		Channel: channel,
		Event:   "unsubscribe",
		Payload: payload,
	}
}

type GateWs struct {
	exchanges.BaseExchange
	orderBooks       map[string]*SymbolOrderBook
	partialOrderBook OrderBook
	errors           map[int]wsex.ExError
}

func (e *GateWs) Init(options wsex.Options) {
	e.BaseExchange.Init()
	e.Option = options
	e.orderBooks = make(map[string]*SymbolOrderBook)
	e.errors = map[int]wsex.ExError{}
	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://api.gateio.ws/ws/v4/"
	}
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.gateio.ws"
	}

}

func (e *GateWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	var (
		channel string
		payload []string
	)
	if isIncremental {
		channel = "spot.order_book_update"
		payload = []string{market.SymbolID, fmt.Sprintf("%dms", speed)}
	} else {
		if level!=5&&level!=10&&level!=20{
			level=20
		}
		channel = "spot.order_book"
		payload = []string{market.SymbolID, fmt.Sprintf("%d", level), fmt.Sprintf("%dms", speed)}
	}
	return e.Subscribe(e.Option.WsHost, channel, payload, sub)
}

func (e *GateWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	return e.Subscribe(e.Option.WsHost, "spot.tickers", []string{market.SymbolID}, sub)
}

func (e *GateWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}
	return e.Subscribe(e.Option.WsHost, "spot.trades", []string{market.SymbolID}, sub)
}

func (e *GateWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *GateWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	table := ""
	switch t {
	case wsex.KLine1Minute:
		table = "1m"
	case wsex.KLine5Minute:
		table = "5m"
	case wsex.KLine15Minute:
		table = "15m"
	case wsex.KLine30Minute:
		table = "30m"
	case wsex.KLine1Hour:
		table = "1h"
	case wsex.KLine4Hour:
		table = "4h"
	case wsex.KLine1Day:
		table = "1d"
	case wsex.KLine1Week:
		table = "7d"
	default:
		{
			err := errors.New("kline does not support this interval")
			return "", err
		}
	}
	var payload = []string{table, market.SymbolID}
	return e.Subscribe(e.Option.WsHost, "spot.candlesticks", payload, sub)
}

func (e *GateWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribeUserData(e.Option.WsHost, "spot.balances", nil, sub)
}

func (e *GateWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	return e.subscribeUserData(e.Option.WsHost, "spot.orders", []string{market.SymbolID}, sub)
}

func (e *GateWs) UnSubscribe(topics string, sub wsex.MessageChan) error {
	stream := UnSubscribeStream(topics, nil)
	conn, err := e.ConnectionMgr.GetConnection(e.Option.WsHost, e.Connect)
	if err != nil {
		return err
	}
	err = e.send(conn, stream)
	if err != nil {
		return err
	}
	conn.Subscribe(sub)
	return nil
}

func (e *GateWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("gate"),
		websocket.SetWsUrl(url),
		websocket.SetProxyUrl(e.Option.ProxyUrl),
		websocket.SetIsAutoReconnect(e.Option.AutoReconnect),
		websocket.SetHeartbeatIntervalTime(time.Second*10),
		websocket.SetReadDeadLineTime(time.Minute*3*2),
		websocket.SetMessageHandler(e.messageHandler),
		websocket.SetErrorHandler(e.errorHandler),
		websocket.SetCloseHandler(e.closeHandler),
		websocket.SetReConnectedHandler(e.reConnectedHandler),
		websocket.SetDisConnectedHandler(e.disConnectedHandler),
		websocket.SetHeartbeatHandler(e.heartbeatHandler),
	)
	return conn, err
}

func (e *GateWs) Subscribe(url, channel string, payload []string, sub wsex.MessageChan) (string, error) {
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}
	err = e.send(conn, SubscribeStream(channel, payload))
	if err != nil {
		return "", err
	}
	conn.Subscribe(sub)
	return channel, nil
}

func (e *GateWs) subscribeUserData(url, channel string, payload []string, sub wsex.MessageChan) (string, error) {
	stream := SubscribeStream(channel, payload)
	s := fmt.Sprintf("channel=%s&event=%s&time=%d", channel, stream.Event, stream.Time)
	singStr, err := utils.HmacSign(utils.SHA512, s, e.Option.SecretKey, false)
	if err != nil {
		return "", err
	}
	var request = make(map[string]string)
	request["method"] = "api_key"
	request["KEY"] = e.Option.AccessKey
	request["SIGN"] = singStr
	stream.Auth = request
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}
	err = e.send(conn, stream)
	if err != nil {
		return "", err
	}
	conn.Subscribe(sub)
	return channel, nil
}

func (e *GateWs) send(conn *exchanges.Connection, stream Stream) (err error) {
	err = conn.SendJsonMessage(stream)
	if err != nil {
		return
	}
	return nil
}

func (e *GateWs) messageHandler(url string, message []byte) {
	var res ResponseEvent
	err := json.Unmarshal(message, &res)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[GateWs] messageHandler unmarshal error:%v", err))
		return
	}
	if res.Error != nil {
		e.errorHandler(url, fmt.Errorf("[GateWs] messageHandler response error:%v", res.Error))
		return
	}
	switch res.Channel {
	case "spot.order_book_update":

		e.handleIncrementalDepth(url, message)
	case "spot.tickers":

		e.handleTicker(url, message)
	case "spot.trades":

		e.handleTrade(url, message)
	case "spot.candlesticks":

		e.handleKLine(url, message)
	case "spot.orders":
		e.handleOrder(url, message)
	case "spot.balances":

		e.handleBalance(url, message)
	case "spot.order_book":
		e.handleDepth(url, message)
	case "spot.pong":
		return

	default:
		e.errorHandler(url, fmt.Errorf("[gateWs] messageHandler - not support this channel :%v", res.Channel))
	}

}

func (e *GateWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)

}

func (e *GateWs) disConnectedHandler(url string, err error) {
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		delete(e.orderBooks, url)
		e.partialOrderBook = OrderBook{}
	})
}

func (e *GateWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *GateWs) closeHandler(url string) {
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
	})
}

func (e *GateWs) heartbeatHandler(url string) {
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	var ping = Stream{
		Time:    time.Now().Unix(),
		Channel: "spot.ping",
	}
	data, _ := json.Marshal(ping)
	conn.SendMessage(data)
}

func (e *GateWs) handleIncrementalDepth(url string, message []byte) {
	type WsIncrementalDepth struct {
		ResponseEvent
		Result RawOrderBook `rest:"result"`
	}
	var data WsIncrementalDepth
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	err := restJson.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[GateWs] handleIncrementalDepth - message Unmarshal to RawOrderBook error:%v", err))
		return
	}

	market, err := e.GetMarketByID(data.Result.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}

	symbolOrderBook, ok := e.orderBooks[url]
	if !ok || symbolOrderBook == nil {
		_ = e.getSnapshotOrderBook(url, market, &SymbolOrderBook{})
		return
	}

	fullOrders, ok := (*e.orderBooks[url])[market.Symbol]
	if !ok {
		_ = e.getSnapshotOrderBook(url, market, &SymbolOrderBook{})
		return
	}
	if fullOrders != nil {
		first := data.Result.FirstUpdateID <= fullOrders.LastUpdateID && data.Result.LastUpdateID >= fullOrders.LastUpdateID
		if first || data.Result.FirstUpdateID == fullOrders.LastUpdateID+1 {
			fullOrders.OrderBook = data.Result.parseOrderBook(market.Symbol)
			((*e.orderBooks[url])[market.Symbol]).LastUpdateID = data.Result.LastUpdateID
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: fullOrders.OrderBook})
		}
	} else if data.Result.LastUpdateID < fullOrders.LastUpdateID {
		log.Printf("[GateWs] handleIncrementalDepth - recv old update data\n")
	} else {
		delete(*symbolOrderBook, market.Symbol)
		err := wsex.ExError{Code: wsex.ErrInvalidDepth,
			Message: fmt.Sprintf("[GateWs] handleIncrementalDepth - recv dirty data, new.FirstUpdateID: %v != old.LastUpdateID: %v ", data.Result.FirstUpdateID, fullOrders.LastUpdateID+1),
			Data:    map[string]interface{}{"symbol": fullOrders.Symbol}}
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: err})
	}
}
func (e *GateWs) handleTicker(url string, message []byte) {
	type tick struct {
		ResponseEvent
		Result Ticker `json:"result"`
	}
	var data tick
	err := json.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[gateWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Result.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}
	ticker := data.Result.parseTicker(market.Symbol)
	ticker.Timestamp = time.Duration(data.ResponseEvent.Time)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: ticker})
}

func (e *GateWs) handleTrade(url string, message []byte) {
	type Traders struct {
		ResponseEvent
		Result Trade `json:"result"`
	}
	var data Traders
	err := json.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[gateWs] handleTrade - message Unmarshal to trader error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Result.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}
	trade := data.Result.parseTrade(market.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: trade})
}
func (e *GateWs) handleKLine(url string, message []byte) {
	type wskline struct {
		ResponseEvent
		Result WsKline `json:"result"`
	}
	var data wskline
	err := json.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[gateWs] handleKLine - message Unmarshal to kline error:%v", err))
		return
	}
	if data.Result.IntSymbol != "" {
		k := strings.Split(data.Result.IntSymbol, "_")
		symbol := fmt.Sprintf("%s/%s", k[1], k[2])
		var table wsex.KLineType
		switch k[0] {
		case "1m":
			table = wsex.KLine1Minute
		case "5m":
			table = wsex.KLine5Minute
		case "15m":
			table = wsex.KLine15Minute
		case "30m":
			table = wsex.KLine30Minute
		case "1h":
			table = wsex.KLine1Hour
		case "4h":
			table = wsex.KLine4Hour
		case "1d":
			table = wsex.KLine1Day
		case "7d":
			table = wsex.KLine1Week
		}
		var kline = wsex.KLine{
			Symbol:    symbol,
			Type:      table,
			Timestamp: time.Duration(utils.SafeParseFloat(data.Result.Timestamp)),
			Open:      utils.SafeParseFloat(data.Result.Open),
			Close:     utils.SafeParseFloat(data.Result.Open),
			High:      utils.SafeParseFloat(data.Result.High),
			Low:       utils.SafeParseFloat(data.Result.Low),
			Volume:    utils.SafeParseFloat(data.Result.Volume),
		}
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgKLine, Data: kline})
	}
}
func (e *GateWs) handleOrder(url string, message []byte) {
	type WsOrder struct {
		ResponseEvent
		Result []Order `rest:"result"`
	}
	if strings.Contains(string(message), "\"status\":\"success\"") {
		return
	}
	var data WsOrder
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	err := restJson.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[gateWs] handleOrder - message Unmarshal to order error:%v", err))
		return
	}
	for _, each := range data.Result {
		if each.Symbol == "" {
			continue
		}
		market, err := e.GetMarketByID(each.Symbol)
		if err != nil {
			e.errorHandler(url, err)
			return
		}
		order := each.parserOrder(market.Symbol)
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
	}
}
func (e *GateWs) handleBalance(url string, message []byte) {
	type WsRest struct {
		ResponseEvent
		Result []WsBalance `json:"result"`
	}
	if strings.Contains(string(message), "\"status\":\"success\"") {
		return
	}
	var data WsRest
	err := json.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[gateWs] handleBalance - message Unmarshal to Balance error:%v", err))
		return
	}
	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	balances.UpdateTime = time.Duration(data.ResponseEvent.Time)
	balance := wsex.Balance{
		Asset:     strings.ToUpper(data.Result[0].Currency),
		Available: utils.SafeParseFloat(data.Result[0].Available),
		Frozen:    utils.SafeParseFloat(data.Result[0].Total) - utils.SafeParseFloat(data.Result[0].Available),
	}
	balances.Balances[balance.Asset] = balance
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}
func (e *GateWs) handleDepth(url string, message []byte) {
	type WsDepth struct {
		ResponseEvent
		Result RawOrderBook `json:"result"`
	}
	var data WsDepth
	err := json.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[gateWs] handleDepth - message Unmarshal to Depth error:%v", err))
		return
	}
	if e.partialOrderBook.LastUpdateID > data.Result.LastUpdateID {
		return
	} else {
		e.partialOrderBook.LastUpdateID = data.Result.LastUpdateID
	}
	market, err := e.GetMarketByID(data.Result.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}
	e.partialOrderBook.Bids = wsex.Depth{}
	e.partialOrderBook.Asks = wsex.Depth{}
	e.partialOrderBook.OrderBook = data.Result.parseOrderBook(market.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: e.partialOrderBook.OrderBook})
}

func (e *GateWs) getSnapshotOrderBook(url string, market wsex.Market, symbolOrderBook *SymbolOrderBook) (err error) {
	time.Sleep(time.Second)
	var response = RawOrderBook{}
	client := resty.New()
	reqUrl := fmt.Sprintf("%s/api/v4/spot/order_book?currency_pair=%s&limit=100&with_id=true", e.Option.RestHost, market.SymbolID)
	_, err = client.R().SetHeader("Accept", "application/json").SetHeader("Content-Type", "application/json").SetResult(&response).Get(reqUrl)
	if err != nil {
		return fmt.Errorf("[GateWs] getSnapshotOrderBook - request url %s error:%v", reqUrl, err)
	}
	if response.ID == 0 {
		return fmt.Errorf("[GateWs] getSnapshotOrderBook - request url %s no data", reqUrl)
	}
	var orderBook OrderBook
	orderBook.LastUpdateID = response.ID
	orderBook.OrderBook = response.parseOrderBook(market.Symbol)
	(*symbolOrderBook)[market.Symbol] = &orderBook
	e.orderBooks[url] = symbolOrderBook
	return
}
