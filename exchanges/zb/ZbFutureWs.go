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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/utils"

	"github.com/shiguantian/wsex"

	"github.com/shiguantian/wsex/exchanges/websocket"
)

type SubTopic struct {
	Topic       string
	Symbol      string
	MessageType wsex.MessageType
	KLineType   wsex.KLineType
}

type ZbFutureWs struct {
	exchanges.BaseExchange
	orderBooks   map[string]*SymbolOrderBook
	subTopicInfo map[string]SubTopic
	errors       map[int]wsex.ExError
	isLogin      bool
	loginChan    chan struct{}
	loginLock    sync.Mutex
	accountType  wsex.FutureAccountType
	contractType wsex.ContractType
	futuresKind  wsex.FuturesKind
}

func (e *ZbFutureWs) Init(option wsex.Options) {
	e.BaseExchange.Init()
	e.Option = option
	e.orderBooks = make(map[string]*SymbolOrderBook)
	e.errors = map[int]wsex.ExError{}
	e.loginChan = make(chan struct{})
	e.isLogin = false
	e.subTopicInfo = make(map[string]SubTopic)
	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://futures.zb.com/ws"
	}
}

func (e *ZbFutureWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	var topic string = ""
	if isIncremental {
		t, err := e.getTopicBySymbol("", symbol, ".Depth")
		if err != nil {
			return "", err
		}
		topic = t
	} else {
		t, err := e.getTopicBySymbol("", symbol, ".DepthWhole")
		if err != nil {
			return "", err
		}
		topic = t
	}
	if level != 5 && level != 10 && level != 20 && level != 50 {
		level = 20
	}
	stream := Stream{
		"channel": topic,
		"size":    fmt.Sprintf("%v", level),
	}
	return e.subscribe(fmt.Sprintf("%s/public/v1", e.Option.WsHost), symbol, SubTopic{Topic: topic, Symbol: symbol, MessageType: wsex.MsgOrderBook}, false, stream, sub)
}

func (e *ZbFutureWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("", symbol, ".Trade")
	if err != nil {
		return "", err
	}
	stream := Stream{
		"channel": topic,
		"size":    "5",
		//"futuresAccountType": e.getAccountType(),
	}
	return e.subscribe(fmt.Sprintf("%s/public/v1", e.Option.WsHost), symbol, SubTopic{Topic: topic, Symbol: symbol, MessageType: wsex.MsgTrade}, false, stream, sub)
}

func (e *ZbFutureWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("", symbol, ".Ticker")
	if err != nil {
		return "", err
	}
	stream := Stream{
		"channel": topic,
		//"futuresAccountType": e.getAccountType(),
	}
	return e.subscribe(fmt.Sprintf("%s/public/v1", e.Option.WsHost), symbol, SubTopic{Topic: topic, Symbol: symbol, MessageType: wsex.MsgTicker}, false, stream, sub)
}

func (e *ZbFutureWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *ZbFutureWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	kline := "1M"
	switch t {
	case wsex.KLine1Day:
		kline = "1D"
	case wsex.KLine1Hour:
		kline = "1H"
	case wsex.KLine6Hour:
		kline = "6H"
	case wsex.KLine1Minute:
		kline = "1M"
	case wsex.KLine5Minute:
		kline = "5M"
	case wsex.KLine15Minute:
		kline = "15M"
	case wsex.KLine30Minute:
		kline = "30M"
	default:
		err := errors.New("zb can not support kline interval")
		return "", err
	}
	topic, err := e.getTopicBySymbol("", symbol, fmt.Sprintf(".KLine_%s", kline))
	if err != nil {
		return "", err
	}
	stream := Stream{
		"channel": topic,
		"size":    "50",
		//"futuresAccountType": e.getAccountType(),
	}
	return e.subscribe(fmt.Sprintf("%s/public/v1", e.Option.WsHost), symbol, SubTopic{Topic: topic, Symbol: symbol, MessageType: wsex.MsgKLine, KLineType: t}, false, stream, sub)
}

func (e *ZbFutureWs) SubscribeMarkPrice(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("", symbol, ".mark")
	if err != nil {
		return "", err
	}
	stream := Stream{
		"channel": topic,
		//"futuresAccountType": e.getAccountType(),
	}
	return e.subscribe(fmt.Sprintf("%s/public/v1", e.Option.WsHost), symbol, SubTopic{Topic: topic, Symbol: symbol, MessageType: wsex.MsgMarkPrice}, false, stream, sub)

}

func (e *ZbFutureWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	stream := Stream{
		"channel":            "Fund.assetChange",
		"futuresAccountType": e.getAccountType(),
	}
	return e.subscribe(fmt.Sprintf("%s/private/api/v2", e.Option.WsHost), "", SubTopic{Topic: "Fund.assetChange", Symbol: symbol, MessageType: wsex.MsgBalance}, true, stream, sub)
}

func (e *ZbFutureWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("", symbol, "")
	if err != nil {
		return "", err
	}
	stream := Stream{
		"channel":            "Trade.orderChange",
		"futuresAccountType": e.getAccountType(),
		"symbol":             topic,
	}

	return e.subscribe(fmt.Sprintf("%s/private/api/v2", e.Option.WsHost), topic, SubTopic{Topic: "Trade.orderChange", Symbol: symbol, MessageType: wsex.MsgOrder}, true, stream, sub)
}

func (e *ZbFutureWs) SubscribePositions(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol("", symbol, "")
	if err != nil {
		return "", err
	}
	stream := Stream{
		"channel":            "Positions.change",
		"futuresAccountType": e.getAccountType(),
		"symbol":             topic,
	}

	return e.subscribe(fmt.Sprintf("%s/private/api/v2", e.Option.WsHost), topic, SubTopic{Topic: "Positions.change", Symbol: symbol, MessageType: wsex.MsgPositions}, true, stream, sub)
}

func (e *ZbFutureWs) UnSubscribe(topic string, sub wsex.MessageChan) error {
	topicInfo, ok := e.subTopicInfo[topic] //ok是看当前key是否存在返回布尔，value返回对应key的值
	if ok {
		delete(e.subTopicInfo, topic)
		stream := Stream{"channel": topic, "action": "unsubscribe"}
		url := fmt.Sprintf("%s/private/api/v2", e.Option.WsHost)
		switch topicInfo.MessageType {
		case wsex.MsgKLine, wsex.MsgTicker, wsex.MsgOrderBook, wsex.MsgTrade:
			url = fmt.Sprintf("%s/public/v1", e.Option.WsHost)
		}
		conn, err := e.ConnectionMgr.GetConnection(url, nil)
		if err != nil {
			return err
		}
		if err := conn.SendJsonMessage(stream); err != nil {
			return err
		}
		conn.UnSubscribe(sub)
	}
	return nil
}

func (e *ZbFutureWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("ZbFutureWs"),
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

func (e *ZbFutureWs) subscribe(url, symbol string, topic SubTopic, needLogin bool, stream Stream, sub wsex.MessageChan) (string, error) {
	_, ok := e.subTopicInfo[topic.Topic] //ok是看当前key是否存在返回布尔，value返回对应key的值
	if !ok {
		e.subTopicInfo[topic.Topic] = topic
	}

	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}

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
	}
	stream["action"] = "subscribe"
	if err := conn.SendJsonMessage(stream); err != nil {
		return "", err
	}
	conn.Subscribe(sub)
	return topic.Topic, nil
}

func (e *ZbFutureWs) login(conn *exchanges.Connection) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	stream := Stream{
		"action": "login",
	}
	stream["ZB-APIKEY"] = e.Option.AccessKey
	stream["ZB-TIMESTAMP"] = timestamp
	payload := timestamp + exchanges.GET + "login"
	secretKey, err := utils.HashSign(utils.SHA1, e.Option.SecretKey, false)
	if err != nil {
		return err
	}
	signature, err := utils.HmacSign(utils.SHA256, payload, secretKey, true)
	if err != nil {
		return err
	}
	stream["ZB-SIGN"] = signature
	return conn.SendJsonMessage(stream)
}

func (e *ZbFutureWs) getTopicBySymbol(prefix string, symbol string, suffix string) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	topic := fmt.Sprintf("%s%s%s", prefix, market.SymbolID, suffix)
	return topic, nil
}

func (e *ZbFutureWs) send(conn *exchanges.Connection, stream Stream) (err error) {
	if err = conn.SendJsonMessage(stream); err != nil {
		return err
	}
	return nil
}

func (e *ZbFutureWs) messageHandler(url string, message []byte) {
	res := FutureResponseEvent{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbFutureWs] messageHandler unmarshal error:%v", err))
		return
	}

	if len(res.Code) > 0 {
		e.errorHandler(url, e.handleError(res))
		return
	}

	if res.Action == "login" {
		e.isLogin = true
		e.loginChan <- struct{}{}
		return
	}

	if res.Channel != "" {
		if topicInfo, ok := e.subTopicInfo[res.Channel]; ok {
			switch topicInfo.MessageType {
			case wsex.MsgOrderBook:
				if strings.Contains(res.Channel, "DepthWhole") {
					e.handleDepth(url, message, topicInfo)
				} else {
					e.handleIncrementalDepth(url, message, topicInfo)
				}
			case wsex.MsgTicker:
				e.handleTicker(url, message, topicInfo)
			case wsex.MsgTrade:
				e.handleTrade(url, message, topicInfo)
			case wsex.MsgKLine:
				e.handleKLine(url, message, topicInfo)
			case wsex.MsgBalance:
				e.handleBalance(url, message)
			case wsex.MsgOrder:
				e.handleOrder(url, message, topicInfo)
			case wsex.MsgPositions:
				e.handlePositions(url, message, topicInfo)
			case wsex.MsgMarkPrice:
				e.handleMarkPrice(url, message, topicInfo)
			}
		}
	}
}

func (e *ZbFutureWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
}

func (e *ZbFutureWs) disConnectedHandler(url string, err error) {
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		delete(e.orderBooks, url)
		e.isLogin = false
	})
}

func (e *ZbFutureWs) closeHandler(url string) {
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
	})
}

func (e *ZbFutureWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *ZbFutureWs) heartbeatHandler(url string) {
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	conn.SendJsonMessage(Stream{
		"action": "ping",
	})
}

func (e *ZbFutureWs) handleDepth(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Data struct {
			Asks wsex.RawDepth `json:"asks"`
			Bids wsex.RawDepth `json:"bids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, err)
		return
	}
	orderBook := wsex.OrderBook{
		Symbol: topicInfo.Symbol,
	}
	for _, ask := range response.Data.Asks {
		if item, err := ask.ParseRawDepthItem(); err == nil {
			orderBook.Asks = append(orderBook.Asks, item)
		}
	}
	for _, bid := range response.Data.Bids {
		if item, err := bid.ParseRawDepthItem(); err == nil {
			orderBook.Bids = append(orderBook.Bids, item)
		}
	}
	sort.Reverse(orderBook.Bids)
	sort.Sort(orderBook.Asks)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: orderBook})
}

func (e *ZbFutureWs) handleIncrementalDepth(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Type string `json:"type"`
		Data struct {
			Asks wsex.RawDepth `json:"asks"`
			Bids wsex.RawDepth `json:"bids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, err)
		return
	}
	orderBook := &OrderBook{}
	orderBook.Symbol = topicInfo.Symbol
	for _, ask := range response.Data.Asks {
		if item, err := ask.ParseRawDepthItem(); err == nil {
			orderBook.Asks = append(orderBook.Asks, item)
		}
	}
	for _, bid := range response.Data.Bids {
		if item, err := bid.ParseRawDepthItem(); err == nil {
			orderBook.Bids = append(orderBook.Bids, item)
		}
	}
	sort.Reverse(orderBook.Bids)
	sort.Sort(orderBook.Asks)
	symbolOrderBook, ok := e.orderBooks[url]
	if !ok {
		symbolOrderBook = &SymbolOrderBook{}
		e.orderBooks[url] = symbolOrderBook
	}
	if len(response.Type) > 0 {
		(*symbolOrderBook)[topicInfo.Symbol] = orderBook
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: orderBook.OrderBook})
	} else {
		fullOrderBook, ok := (*symbolOrderBook)[topicInfo.Symbol]
		if !ok || fullOrderBook == nil {
			return
		}
		fullOrderBook.OrderBookUpdate(response.Data.Asks, response.Data.Bids)
		if fullOrderBook.OrderBook.Asks.Len() == 0 || fullOrderBook.OrderBook.Bids.Len() == 0 {
			fmt.Println("234")
		}
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: fullOrderBook.OrderBook})
	}
}

func (o *OrderBook) OrderBookUpdate(asks wsex.RawDepth, bids wsex.RawDepth) {
	o.Bids = o.Bids.Update(bids, true)
	o.Asks = o.Asks.Update(asks, false)
}

func (e *ZbFutureWs) handleTicker(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Data FutureTicker `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, err)
		return
	}
	ticker := response.Data.parseTicker()
	ticker.Symbol = topicInfo.Symbol
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: ticker})
}

func (e *ZbFutureWs) handleTrade(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Data []FutureTrade `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, err)
		return
	}
	if len(response.Data) == 0 {
		return
	}
	trade := response.Data[0].parseTrade()
	trade.Symbol = topicInfo.Symbol
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: trade})
}

func (e *ZbFutureWs) handleKLine(url string, message []byte, topicInfo SubTopic) {
	var data = FutureKLine{}
	if err := json.Unmarshal(message, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, ele := range data.Data {
		kline := wsex.KLine{
			Symbol:    topicInfo.Symbol,
			Timestamp: time.Duration(ele[5]),
			Type:      topicInfo.KLineType,
			Open:      ele[0],
			Close:     ele[3],
			High:      ele[1],
			Low:       ele[2],
			Volume:    ele[4],
		}
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgKLine, Data: kline})
	}
}

func (e *ZbFutureWs) handleMarkPrice(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, err)
		return
	}
	markPrice := wsex.MarkPrice{
		Price:  response.Data,
		Symbol: topicInfo.Symbol,
	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgMarkPrice, Data: markPrice})
}

func (e *ZbFutureWs) handleBalance(url string, message []byte) {
	var response struct {
		Data FutureBalance `json:"data"`
	}
	wsJson := jsoniter.Config{TagKey: "ws"}.Froze()
	if err := wsJson.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbFutureWs] handleBalance - message Unmarshal to balance error:%v", err))
		return
	}

	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	balances.Balances[strings.ToUpper(response.Data.Currency)] = response.Data.parseBalance()
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}

func (e *ZbFutureWs) handleOrder(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Data FutureOrder `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbFutureWs] handleOrder - message Unmarshal to handleOrder error:%v", err))
		return
	}
	order := response.Data.parseOrder(topicInfo.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
}

func (e *ZbFutureWs) handlePositions(url string, message []byte, topicInfo SubTopic) {
	var response struct {
		Data FuturePosition `json:"data"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbFutureWs] handlePositions - message Unmarshal to handleOrder error:%v", err))
		return
	}
	market, err := e.GetMarketByID(response.Data.Symbol)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[ZbFutureWs] handlePositions - can not find market:%v", err))
	}

	positions := response.Data.parsePositions(market.BaseID, market.Symbol)
	var futurePosition = wsex.FuturePositonsUpdate{
		Positons: make([]wsex.FuturePositons, 0),
	}
	futurePosition.Symbol = market.Symbol
	futurePosition.Positons = append(futurePosition.Positons, positions)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgPositions, Data: futurePosition})
}

func (e *ZbFutureWs) handleError(res FutureResponseEvent) wsex.ExError {
	return wsex.ExError{
		Message: res.Code,
	}
}

func (e *ZbFutureWs) getAccountType() string {
	accountType := 1
	if e.accountType == wsex.CoinMargin {
		accountType = 2
	}
	return strconv.Itoa(accountType)
}

func (e *ZbFutureWs) GetMarketByID(symbolID string) (wsex.Market, error) {
	symbolID = strings.ToUpper(symbolID)
	for _, market := range e.Option.Markets {
		//The symbol format returned by ZB websocket is btcusdt,Inconsistent format
		// sID := fmt.Sprintf("%s%s", market.BaseID, market.QuoteID)
		// sID = strings.ToUpper(sID)
		if market.SymbolID == symbolID {
			return market, nil
		}
	}
	return wsex.Market{}, errors.New(fmt.Sprintf("%v market not found", symbolID))
}
