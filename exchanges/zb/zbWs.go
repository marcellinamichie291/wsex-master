/*
@Time : 2021/2/24 10:23 上午
@Author : shiguantian
@File : zb
@Software: GoLand
*/
package zb

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shiguantian/wsex/exchanges"

	"github.com/shiguantian/wsex"

	"github.com/shiguantian/wsex/exchanges/websocket"
	. "github.com/shiguantian/wsex/utils"
)

type Stream map[string]string

func (s Stream) subscribe() {
	s["event"] = "addChannel"
}

func (s Stream) unSubscribe() {
	s["event"] = "removeChannel"
}

func (s Stream) set(k, v string) {
	s[k] = v
}

type ZbWs struct {
	exchanges.BaseExchange
	orderBooks map[string]*SymbolOrderBook
	errors     map[int]wsex.ExError
}

func (e *ZbWs) Init(option wsex.Options) {
	e.BaseExchange.Init()
	e.Option = option
	e.orderBooks = make(map[string]*SymbolOrderBook)
	e.errors = map[int]wsex.ExError{}

	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://api.zb.com/websocket"
	}
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.zb.com"
	}
}

func (e *ZbWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "_quick_depth")
	if err != nil {
		return "", err
	}
	if level != 5 && level != 10 && level != 20 {
		level = 20
	}
	stream := Stream{
		"channel": topic,
		"length":  fmt.Sprintf("%v", level),
	}
	market, _ := e.GetMarket(symbol)
	url := fmt.Sprintf("%s/%s", e.Option.WsHost, market.BaseID)
	return e.subscribe(url, topic, false, stream, sub)
}

func (e *ZbWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "_trades")
	if err != nil {
		return "", err
	}
	stream := Stream{"channel": topic}
	return e.subscribe(e.Option.WsHost, topic, false, stream, sub)
}

func (e *ZbWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "_ticker")
	if err != nil {
		return "", err
	}
	stream := Stream{"channel": topic}
	return e.subscribe(e.Option.WsHost, topic, false, stream, sub)
}

func (e *ZbWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *ZbWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *ZbWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	topic := "push_user_incr_asset"
	topic = strings.ToLower(topic)
	stream := Stream{
		"channel":   topic,
		"accesskey": e.Option.AccessKey,
		"event":     "addChannel",
	}
	return e.subscribe(e.Option.WsHost, topic, true, stream, sub)
}

func (e *ZbWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	market, err := e.getTopicBySymbol(symbol, "default")
	if err != nil {
		return "", err
	}
	topic := "push_user_incr_record"
	stream := Stream{
		"channel":   topic,
		"accesskey": e.Option.AccessKey,
		"market":    market,
		"event":     "addChannel",
	}

	return e.subscribe(e.Option.WsHost, topic, true, stream, sub)
}

func (e *ZbWs) UnSubscribe(topic string, sub wsex.MessageChan) error {
	stream := Stream{"event": topic}
	stream.unSubscribe()
	conn, err := e.ConnectionMgr.GetConnection(e.Option.WsHost, nil)
	if err != nil {
		return err
	}
	if err := conn.SendJsonMessage(stream); err != nil {
		return err
	}
	conn.UnSubscribe(sub)

	return nil
}

func (e *ZbWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("ZbWs"),
		websocket.SetWsUrl(url),
		websocket.SetProxyUrl(e.Option.ProxyUrl),
		websocket.SetIsAutoReconnect(e.Option.AutoReconnect),
		websocket.SetEnableCompression(false),
		websocket.SetHeartbeatIntervalTime(time.Second*5),
		websocket.SetReadDeadLineTime(time.Second*10),
		websocket.SetMessageHandler(e.messageHandler),
		websocket.SetErrorHandler(e.errorHandler),
		websocket.SetCloseHandler(e.closeHandler),
		websocket.SetReConnectedHandler(e.reConnectedHandler),
		websocket.SetDisConnectedHandler(e.disConnectedHandler),
		websocket.SetHeartbeatHandler(e.heartbeatHandler),
	)
	return conn, err
}

func (e *ZbWs) subscribe(url, topic string, needSign bool, stream Stream, sub wsex.MessageChan) (string, error) {
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}

	if needSign {
		signData, err := e.sign(stream)
		if err != nil {
			return "", err
		}
		stream.set("sign", signData)
	}
	stream.subscribe()
	if err := e.send(conn, stream); err != nil {
		return "", err
	}
	conn.Subscribe(sub)
	return topic, nil
}

func (e *ZbWs) getTopicBySymbol(symbol, suffix string) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	topic := fmt.Sprintf("%s%s%s", market.BaseID, market.QuoteID, suffix)
	return strings.ToLower(topic), nil
}

func (e *ZbWs) send(conn *exchanges.Connection, stream Stream) (err error) {
	if err = conn.SendJsonMessage(stream); err != nil {
		return err
	}

	return nil
}

func (e *ZbWs) messageHandler(url string, message []byte) {
	if string(message) == "pong" {
		return
	}

	res := ResponseEvent{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbWs] messageHandler unmarshal error:%v", err))
		return
	}

	if res.Code != 0 && res.Code != 1000 {
		e.errorHandler(url, e.handleError(res))
		return
	}

	if res.Channel == "" {
		e.errorHandler(url, fmt.Errorf("[ZbWs] messageHandler - message no channel field, %v", res))
		return
	}

	if strings.Contains(res.Channel, "depth") {
		e.handleDepth(url, message)
	} else if strings.Contains(res.Channel, "ticker") {
		e.handleTicker(url, message)
	} else if strings.Contains(res.Channel, "trades") {
		e.handleTrade(url, message)
	} else if strings.Contains(res.Channel, "record") {
		e.handleOrder(url, message)
	} else if strings.Contains(res.Channel, "asset") {
		e.handleBalance(url, message)
	} else {
		e.errorHandler(url, fmt.Errorf("[ZbWs] messageHandler - not support this channel :%v", res.Channel))
	}

}

func (e *ZbWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
}

func (e *ZbWs) disConnectedHandler(url string, err error) {
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		delete(e.orderBooks, url)
	})
}

func (e *ZbWs) closeHandler(url string) {
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
	})
}

func (e *ZbWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *ZbWs) heartbeatHandler(url string) {
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	conn.SendMessage([]byte("ping"))
}

func (e *ZbWs) handleDepth(url string, message []byte) {
	//ZB doesn't support incremental push, push top [level] data each time.
	data := RawOrderBook{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbWs] handleDepth - message Unmarshal to RawOrderBook error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}

	symbolOrderBook, ok := e.orderBooks[url]
	if !ok {
		symbolOrderBook = &SymbolOrderBook{}
		e.orderBooks[url] = symbolOrderBook
	}
	fullOrderBook, ok := (*symbolOrderBook)[market.Symbol]
	if !ok {
		fullOrderBook := &OrderBook{}
		fullOrderBook.update(data)
		fullOrderBook.Symbol = market.Symbol
		(*symbolOrderBook)[market.Symbol] = fullOrderBook
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: fullOrderBook.OrderBook})
	} else if fullOrderBook != nil {
		if data.LastTime >= fullOrderBook.LastTime {
			fullOrderBook.update(data)
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: fullOrderBook.OrderBook})
		} else {
			delete(*symbolOrderBook, market.Symbol)
			err := wsex.ExError{Code: wsex.ErrInvalidDepth,
				Message: fmt.Sprintf("[ZbWs] handleDepth - recv dirty data, new.LastTime: %v < old.LastTime: %v ", data.LastTime, fullOrderBook.LastTime),
				Data:    map[string]interface{}{"symbol": fullOrderBook.Symbol}}
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgError, Data: err})
			return
		}
	}
}

func (e *ZbWs) handleTicker(url string, message []byte) {
	data := TickerRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}

	ticker := data.parseTicker()
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: ticker})
}

func (e *ZbWs) handleTrade(url string, message []byte) {
	data := TradeRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbWs] handleTrade - message Unmarshal to trade error:%v", err))
		return
	}
	if len(data.Trade) == 0 {
		return
	}

	trade := data.Trade[0].parseTrade()
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: trade})
}

func (e *ZbWs) handleBalance(url string, message []byte) {
	data := Balances{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbWs] handleBalance - message Unmarshal to balance error:%v", err))
		return
	}

	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	balances.UpdateTime = time.Duration(data.Timestamp)
	for _, item := range data.Balances {
		balance := item.parseBalance()
		balances.Balances[balance.Asset] = balance
	}

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}

func (e *ZbWs) handleOrder(url string, message []byte) {
	data := Order{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbWs] handleOrder - message Unmarshal to handleOrder error:%v", err))
		return
	}

	if len(data.Record) < 13 {
		e.errorHandler(url, fmt.Errorf("[ZbWs] handleOrder - order data not match data:%v", data))
		return
	}

	symbol := strings.TrimSuffix(data.Symbol, "default")
	market, _ := e.GetMarketByID(symbol)
	var (
		price, amount, filled, cost, createTime, oType, status float64
	)

	order := wsex.Order{}
	SafeAssign(data.Record[0], &order.ID)
	SafeAssign(data.Record[1], &price)
	SafeAssign(data.Record[2], &amount)
	SafeAssign(data.Record[3], &filled)
	SafeAssign(data.Record[4], &cost)
	order.Price = strconv.FormatFloat(price, 'f', market.PricePrecision, 64)
	order.Amount = strconv.FormatFloat(amount, 'f', market.AmountPrecision, 64)
	order.Filled = fmt.Sprintf("%v", filled)
	order.Cost = fmt.Sprintf("%v", cost)
	SafeAssign(data.Record[5], &oType)
	order.Side = parseSide(int(oType))
	SafeAssign(data.Record[6], &createTime)
	order.CreateTime = time.Duration(createTime)
	SafeAssign(data.Record[7], &status)
	order.Status = parseStatus(int(status), filled)
	order.Symbol = market.Symbol

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
}

func (e *ZbWs) sign(params map[string]string) (string, error) {
	payload, _ := json.Marshal(params)
	secretKey, err := HashSign(SHA1, e.Option.SecretKey, false)
	if err != nil {
		return "", err
	}
	sign, err := HmacSign(Md5, string(payload), secretKey, false)
	if err != nil {
		return "", err
	}
	return sign, nil
}

func (e *ZbWs) handleError(res ResponseEvent) wsex.ExError {
	return wsex.ExError{}
}

func (e *ZbWs) GetMarketByID(symbolID string) (wsex.Market, error) {
	symbolID = strings.ToUpper(symbolID)
	for _, market := range e.Option.Markets {
		//The symbol format returned by ZB websocket is btcusdt,Inconsistent format
		sID := fmt.Sprintf("%s%s", market.BaseID, market.QuoteID)
		sID = strings.ToUpper(sID)
		if sID == symbolID {
			return market, nil
		}
	}
	return wsex.Market{}, errors.New(fmt.Sprintf("%v market not found", symbolID))
}
