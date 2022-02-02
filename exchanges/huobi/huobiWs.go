package huobi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/exchanges/websocket"
	. "github.com/shiguantian/wsex/utils"
)

type Stream map[string]string

type SubTopic struct {
	Topic        string
	Symbol       string
	MessageType  wsex.MessageType
	LastUpdateID float64
}

type HuobiWs struct {
	exchanges.BaseExchange
	orderBooks   map[string]*SymbolOrderBook
	subTopicInfo map[string]SubTopic
	errors       map[int]wsex.ExError
	loginLock    sync.Mutex
	loginChan    chan struct{}
	isLogin      bool
}

func (e *HuobiWs) Init(option wsex.Options) {
	e.BaseExchange.Init()
	e.Option = option
	e.orderBooks = make(map[string]*SymbolOrderBook)
	e.subTopicInfo = make(map[string]SubTopic)
	e.errors = map[int]wsex.ExError{}
	e.loginLock = sync.Mutex{}
	e.loginChan = make(chan struct{})
	e.isLogin = false
	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://api.huobi.pro/ws"
	}
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.huobi.pro"
	}
}

func (e *HuobiWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	suffix := ""
	url := e.Option.WsHost
	if !isIncremental {
		if level != 5 && level != 10 && level != 20 {
			level = 20
		}
		suffix = fmt.Sprintf(".mbp.refresh.%v", level)
	} else {
		if level != 5 && level != 20 && level != 150 {
			level = 150
		}
		suffix = fmt.Sprintf(".mbp.%v", level)
		url = "wss://api.huobi.pro/feed"
	}
	topic, err := e.getTopicBySymbol("market.", symbol, suffix)
	if err != nil {
		return "", err
	}
	return e.subscribe(url, topic, symbol, wsex.MsgOrderBook, false, sub)
}

func (e *HuobiWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("market.", symbol, ".detail")
	if err != nil {
		return "", err
	}
	return e.subscribe(e.Option.WsHost, topic, symbol, wsex.MsgTicker, false, sub)
}

func (e *HuobiWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("market.", symbol, ".trade.detail")
	if err != nil {
		return "", err
	}
	return e.subscribe(e.Option.WsHost, topic, symbol, wsex.MsgTrade, false, sub)
}

func (e *HuobiWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *HuobiWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	table := ""
	switch t {
	case wsex.KLine1Minute:
		table = "1min"
	case wsex.KLine5Minute:
		table = "5min"
	case wsex.KLine15Minute:
		table = "15min"
	case wsex.KLine30Minute:
		table = "30min"
	case wsex.KLine1Hour:
		table = "60min"
	case wsex.KLine4Hour:
		table = "4hour"
	case wsex.KLine1Day:
		table = "1day"
	case wsex.KLine1Week:
		table = "1week"
	default:
		{
			err := errors.New("kline does not support this interval")
			return "", err
		}
	}
	topic, err := e.getTopicBySymbol("market.", symbol, ".kline."+table)
	if err != nil {
		return "", err
	}
	return e.subscribe(e.Option.WsHost, topic, symbol, wsex.MsgKLine, false, sub)
}

func (e *HuobiWs) UnSubscribe(event string, sub wsex.MessageChan) error {
	delete(e.subTopicInfo, event)
	conn, err := e.ConnectionMgr.GetConnection(e.Option.WsHost, nil)
	if err != nil {
		return err
	}
	data := map[string]string{
		"unsub": event,
	}
	if err := conn.SendJsonMessage(data); err != nil {
		return err
	}
	conn.UnSubscribe(sub)
	return nil
}

func (e *HuobiWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribe(fmt.Sprintf("%s/v2", e.Option.WsHost), "accounts.update#2", symbol, wsex.MsgBalance, true, sub)
}

func (e *HuobiWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("orders#", symbol, "")
	if err != nil {
		return "", err
	}
	return e.subscribe(fmt.Sprintf("%s/v2", e.Option.WsHost), topic, symbol, wsex.MsgOrder, true, sub)
}

func (e *HuobiWs) getTopicBySymbol(prefix string, symbol string, suffix string) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	topic := fmt.Sprintf("%s%s%s", prefix, market.SymbolID, suffix)
	return strings.ToLower(topic), nil
}

func (e *HuobiWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("Huobi"),
		websocket.SetWsUrl(url),
		websocket.SetIsAutoReconnect(e.Option.AutoReconnect),
		websocket.SetEnableCompression(false),
		websocket.SetReadDeadLineTime(time.Minute),
		websocket.SetMessageHandler(e.messageHandler),
		websocket.SetErrorHandler(e.errorHandler),
		websocket.SetCloseHandler(e.closeHandler),
		websocket.SetReConnectedHandler(e.reConnectedHandler),
		websocket.SetDisConnectedHandler(e.disConnectedHandler),
		websocket.SetDecompressHandler(e.decompressHandler),
	)
	return conn, err
}

func (e *HuobiWs) subscribe(url, topic, symbol string, t wsex.MessageType, needLogin bool, sub wsex.MessageChan) (string, error) {
	_, ok := e.subTopicInfo[topic] //ok是看当前key是否存在返回布尔，value返回对应key的值
	if !ok {
		e.subTopicInfo[topic] = SubTopic{Topic: topic, Symbol: symbol, MessageType: t}
	}

	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}
	var data map[string]string
	if needLogin {
		e.loginLock.Lock()
		defer e.loginLock.Unlock()
		if !e.isLogin {
			if err := e.login(conn); err != nil {
				return "", err
			}
			select {
			case <-e.loginChan:
				break
			case <-time.After(time.Second * 5):
				return "", errors.New("login failed")
			}
		}
		data = map[string]string{
			"action": "sub",
			"ch":     topic,
		}
	} else {
		data = map[string]string{
			"sub": topic,
		}
	}

	if err := conn.SendJsonMessage(data); err != nil {
		return "", err
	}
	conn.Subscribe(sub)

	return topic, nil
}

func (e *HuobiWs) send(url string, data interface{}) (err error) {
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	if err := conn.SendJsonMessage(data); err != nil {
		return err
	}

	return nil
}

func (e *HuobiWs) decompressHandler(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func (e *HuobiWs) messageHandler(url string, message []byte) {
	res := Response{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] messageHandler unmarshal error:%v", err))
		return
	}
	if res.Ping != 0 {
		data := map[string]interface{}{"pong": res.Ping}
		if err := e.send(url, data); err != nil {
			return
		}
		return
	}
	if res.Action != "" {
		if e.handleAction(message, res, url) {
			return
		}
	}

	if res.Topic != "" && res.Code == 0 {
		topicInfo := e.subTopicInfo[res.Topic]
		switch topicInfo.MessageType {
		case wsex.MsgTrade:
			e.handleTrade(url, message, topicInfo)
		case wsex.MsgOrderBook:
			if strings.Contains(res.Topic, "refresh") {
				e.handleDepth(url, message, topicInfo)
			} else {
				e.handleIncrementalDepth(url, message, topicInfo)
			}
		case wsex.MsgKLine:
			e.handleKLine(url, message, topicInfo)
		case wsex.MsgTicker:
			e.handleTicker(url, message, topicInfo)
		case wsex.MsgOrder:
			e.handleOrder(url, message, topicInfo)
		case wsex.MsgBalance:
			e.handleBalance(url, message, topicInfo)
		default:
			e.errorHandler(url, fmt.Errorf("[huobiWs] messageHandler - not support this channel :%v", res.Topic))
		}
	}
	if res.Rep != "" {
		if strings.Contains(res.Rep, "mbp") {
			//get full order book
			topicInfo := e.subTopicInfo[res.Rep]
			e.handleFullDepth(url, message, topicInfo)
		}
	}
}

func (e *HuobiWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
}

func (e *HuobiWs) disConnectedHandler(url string, err error) {
	// clear cache data, Prevent getting dirty data
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		e.isLogin = false
		delete(e.orderBooks, url)
	})
}

func (e *HuobiWs) closeHandler(url string) {
	// clear cache data and the connection
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
	})
}

func (e *HuobiWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *HuobiWs) handleTicker(url string, message []byte, topicInfo SubTopic) {
	var data WsTickerRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	ticker := data.parseWsTicker(topicInfo.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: ticker})
}

func (e *HuobiWs) handleDepth(url string, message []byte, topicInfo SubTopic) {
	var data OrderBookRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	if data.Depth.SeqNum > topicInfo.LastUpdateID {
		res := data.parseOrderBook(topicInfo.Symbol)
		topicInfo.LastUpdateID = data.Depth.SeqNum
		e.subTopicInfo[topicInfo.Topic] = topicInfo
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: res})
	} else {
		err := wsex.ExError{Code: wsex.ErrInvalidDepth,
			Message: fmt.Sprintf("[HuobiWs] handleDepth - recv dirty data, new.SeqNum: %v < old.SeqNum: %v ", data.Depth.PrevSeqNum, topicInfo.LastUpdateID),
			Data:    map[string]interface{}{"symbol": topicInfo.Symbol}}
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: err})
	}
}

func (e *HuobiWs) handleIncrementalDepth(url string, message []byte, topicInfo SubTopic) {
	var data OrderBookRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}

	symbolOrderBook, ok := e.orderBooks[url]
	if !ok {
		symbolOrderBook = &SymbolOrderBook{}
		e.orderBooks[url] = symbolOrderBook
	}
	fullOrderBook, ok := (*symbolOrderBook)[topicInfo.Symbol]
	if !ok {
		e.send(url, map[string]string{"req": topicInfo.Topic})
		(*symbolOrderBook)[topicInfo.Symbol] = &OrderBook{}
		return
	}
	if fullOrderBook != nil {
		if fullOrderBook.SeqNum == data.Depth.PrevSeqNum {
			fullOrderBook.update(data)
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: fullOrderBook.OrderBook})
		} else if fullOrderBook.SeqNum != 0 {
			delete(*symbolOrderBook, topicInfo.Symbol)
			err := wsex.ExError{Code: wsex.ErrInvalidDepth,
				Message: fmt.Sprintf("[HuobiWs] handleIncrementalDepth - recv dirty data, new.PrevSeqNum: %v != old.SeqNum: %v ", data.Depth.PrevSeqNum, fullOrderBook.SeqNum),
				Data:    map[string]interface{}{"symbol": fullOrderBook.Symbol}}
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: err})
		}
	}
}

func (e *HuobiWs) handleFullDepth(url string, message []byte, topicInfo SubTopic) {
	var data OrderBookRes
	restJson := jsoniter.Config{TagKey: "rep"}.Froze()
	if err := restJson.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	ob := OrderBook{}
	ob.update(data)
	symbolOrderBook, ok := e.orderBooks[url]
	if !ok {
		symbolOrderBook = &SymbolOrderBook{}
		e.orderBooks[url] = symbolOrderBook
	}
	(*symbolOrderBook)[topicInfo.Symbol] = &ob
}

func (e *HuobiWs) handleTrade(url string, message []byte, topicInfo SubTopic) {
	type TickerRes struct {
		Ticker struct {
			Data []struct {
				Amount    float64 `json:"amount"`
				Price     float64 `json:"price"`
				Direction string  `json:"direction"`
			}
			Timestamp time.Duration `json:"ts"`
		} `json:"tick"`
		Timestamp time.Duration `json:"ts"`
		Topic     string        `json:"ch"`
	}
	var data TickerRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}

	ticker := wsex.Trade{
		Timestamp: data.Timestamp,
		Symbol:    topicInfo.Symbol,
		Price:     data.Ticker.Data[0].Price,
		Amount:    data.Ticker.Data[0].Amount,
	}
	if data.Ticker.Data[0].Direction == "sell" {
		ticker.Side = wsex.Sell
	} else {
		ticker.Side = wsex.Buy
	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: ticker})
}

func (e *HuobiWs) handleKLine(url string, message []byte, topicInfo SubTopic) {
	var data WsKlineRes
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	kline := data.parseKline(topicInfo.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgKLine, Data: kline})
}

func (e *HuobiWs) handleBalance(url string, message []byte, topicInfo SubTopic) {
	data := Balance{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleBalance - message Unmarshal to balance error:%v", err))
		return
	}

	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	balances.UpdateTime = data.Data.Timestamp
	balance := wsex.Balance{
		Asset:     strings.ToUpper(data.Data.Currency),
		Available: SafeParseFloat(data.Data.Available),
		Frozen:    SafeParseFloat(data.Data.Balance) - SafeParseFloat(data.Data.Available),
	}
	balances.Balances[balance.Asset] = balance

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}

func (e *HuobiWs) handleOrder(url string, message []byte, topicInfo SubTopic) {
	market, err := e.GetMarket(topicInfo.Symbol)
	if err != nil {
		return
	}
	data := OrderRes{}
	wsJson := jsoniter.Config{TagKey: "ws"}.Froze()
	if err := wsJson.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[huobiWs] handleOrder - message Unmarshal to handleOrder error:%v", err))
		return
	}

	order := data.parseOrder(topicInfo.Symbol, market)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
}

func (e *HuobiWs) sign(timeNow string) (string, error) {
	payload := ""
	payload += "accessKey=" + e.Option.AccessKey + "&signatureMethod=HmacSHA256&signatureVersion=2.1&timestamp=" + url.QueryEscape(timeNow)
	plainText := "GET\n"
	plainText += "api.huobi.pro\n"
	plainText += "/ws/v2\n"
	plainText += payload
	println(plainText)
	signature, err := HmacSign(SHA256, plainText, e.Option.SecretKey, true)
	if err != nil {
		return "", err
	}
	return signature, nil
}

func (e *HuobiWs) handleError(res ResponseEvent) wsex.ExError {
	return wsex.ExError{}
}

func (e *HuobiWs) login(conn *exchanges.Connection) error {
	timeNow := time.Now().UTC().Format("2006-01-02T15:04:05")
	signature, err := e.sign(timeNow)
	if err != nil {
		return err
	}
	req := make(map[string]interface{})
	req["action"] = "req"
	req["ch"] = "auth"
	req["params"] = map[string]string{
		"authType":         "api",
		"accessKey":        e.Option.AccessKey,
		"signatureMethod":  "HmacSHA256",
		"signatureVersion": "2.1",
		"timestamp":        timeNow,
		"signature":        signature,
	}
	return conn.SendJsonMessage(req)
}

func (e *HuobiWs) handleAction(message []byte, res Response, url string) bool {
	if res.Action == "ping" {
		var res PingAction
		if err := json.Unmarshal(message, &res); err == nil {
			res.Action = "pong"
			e.send(url, res)
		}
		return true
	} else if res.Action == "req" {
		if res.Topic == "auth" && res.Code == 200 {
			e.isLogin = true
			e.loginChan <- struct{}{}
		}
		return true
	}
	return false
}
