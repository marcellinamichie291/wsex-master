package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/go-resty/resty/v2"
	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/exchanges/websocket"
	"github.com/shiguantian/wsex/utils"
)

type Fstream struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int      `json:"id"`
}

func SubscribeFstream(event ...string) Stream {
	return Stream{
		Method: "SUBSCRIBE",
		Params: append([]string{}, event...),
		Id:     time.Now().Nanosecond(),
	}
}

func UnSubscribeFstream(event ...string) Stream {
	return Stream{
		Method: "UNSUBSCRIBE",
		Params: append([]string{}, event...),
		Id:     time.Now().Nanosecond(),
	}
}

type UserDataFstreamError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (u UserDataFstreamError) Error() string {
	return fmt.Sprintf("UserDataStreamError error, code:%v msg:%v", u.Code, u.Msg)
}

type BinanceFutureWs struct {
	exchanges.BaseExchange
	accountType  wsex.FutureAccountType
	contractType wsex.ContractType
	futuresKind  wsex.FuturesKind

	isIncrementalDepth bool
	isSubUserData      bool
	orderBooks         map[string]*SymbolOrderBook // orderbook's local cache of one symbol
	partialOrderBook   OrderBook                   // Partial Book Depth
	errors             map[int]wsex.ExError
	listenKey          string // listenKey for User Data Streams, including account update,balance update,order update
	listenKeyStop      chan struct{}
}

func (e *BinanceFutureWs) Init(option wsex.Options) {
	e.BaseExchange.Init()
	e.Option = option
	e.orderBooks = make(map[string]*SymbolOrderBook)
	e.errors = map[int]wsex.ExError{
		30040: wsex.ExError{Code: wsex.ErrChannelNotExist},
		30008: wsex.ExError{Code: wsex.ErrAuthFailed},
		30013: wsex.ExError{Code: wsex.ErrAuthFailed},
		30027: wsex.ExError{Code: wsex.ErrAuthFailed},
		30041: wsex.ExError{Code: wsex.ErrAuthFailed},
	}
	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://fstream.binance.com/ws"
	}
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://fapi.binance.com"
	}
	e.listenKeyStop = make(chan struct{})
	e.isSubUserData = false
}

func (e *BinanceFutureWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	e.RwLock.Lock()
	if !isIncremental {
		if e.partialOrderBook.Symbol != "" {
			return "", errors.New("binance instance can only obtain one symbol partial order book at the same time")
		}
		e.partialOrderBook.Symbol = symbol
	}
	e.RwLock.Unlock()
	e.isIncrementalDepth = isIncremental
	var suffix = "depth"
	if level > 20 {
		level = 20
	}
	if !isIncremental && level > 0 && (level == 5 || level == 10 || level == 20) {
		suffix = fmt.Sprintf("%s%d", suffix, level)
	}
	if speed > 0 {
		suffix = fmt.Sprintf("%s@%dms", suffix, speed)
	}
	topic, err := e.getTopicBySymbol(symbol, suffix)
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceFutureWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "aggTrade")
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceFutureWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "ticker")
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceFutureWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	topic := "!ticker@arr"
	topic = strings.ToLower(topic)
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceFutureWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	kt := parseKLienType(t)
	topic, err := e.getTopicBySymbol(symbol, fmt.Sprintf("kline_%s", kt))
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceFutureWs) SubscribeMarkPrice(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "markPrice@1s")
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceFutureWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribeUserData(sub)
}

func (e *BinanceFutureWs) SubscribePositions(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribeUserData(sub)
}

func (e *BinanceFutureWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribeUserData(sub)
}

func (e *BinanceFutureWs) UnSubscribe(event string, sub wsex.MessageChan) error {
	conn, err := e.ConnectionMgr.GetConnection(e.Option.WsHost, nil)
	if err != nil {
		return err
	}
	if err := e.send(conn, UnSubscribeFstream(event)); err != nil {
		return err
	}

	conn.UnSubscribe(sub)
	return nil
}

func (e *BinanceFutureWs) getTopicBySymbol(symbol, suffix string) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	var topic string
	if e.contractType == "Futures" {
		topic = fmt.Sprintf("%s@%s", strings.ToLower(market.SymbolID), suffix)
	} else {
		topic = fmt.Sprintf("%s%s@%s", strings.ToLower(market.BaseID), strings.ToLower(market.QuoteID), suffix)
	}

	return topic, nil
}

func (e *BinanceFutureWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("binance"),
		websocket.SetWsUrl(url),
		websocket.SetProxyUrl(e.Option.ProxyUrl),
		websocket.SetIsAutoReconnect(e.Option.AutoReconnect),
		websocket.SetHeartbeatIntervalTime(time.Second*10),
		websocket.SetReadDeadLineTime(time.Minute*3*2), // binance's heartbeat interval is 3 minutes
		websocket.SetMessageHandler(e.messageHandler),
		websocket.SetErrorHandler(e.errorHandler),
		websocket.SetCloseHandler(e.closeHandler),
		websocket.SetReConnectedHandler(e.reConnectedHandler),
		websocket.SetDisConnectedHandler(e.disConnectedHandler),
		websocket.SetHeartbeatHandler(e.heartbeatHandler),
	)
	return conn, err
}

func (e *BinanceFutureWs) subscribe(url, topic string, sub wsex.MessageChan) (string, error) {
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}
	if err := e.send(conn, SubscribeFstream(topic)); err != nil {
		return "", err
	}
	conn.Subscribe(sub)
	return topic, nil
}

func (e *BinanceFutureWs) subscribeUserData(sub wsex.MessageChan) (string, error) {
	e.RwLock.Lock()
	e.isSubUserData = true
	defer e.RwLock.Unlock()
	if e.listenKey != "" {
		// Because balance and order share the same stream, so just subscribe once.
		url := fmt.Sprintf("%s/%s", e.Option.WsHost, e.listenKey)
		conn, err := e.ConnectionMgr.GetConnection(url, nil)
		if err == nil {
			conn.Subscribe(sub)
		}
		return e.listenKey, err
	}
	var err error
	e.listenKey, err = e.createListenKey()
	if err != nil {
		return e.listenKey, err
	}
	e.listenKeyStop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Minute * 30)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				e.keepAliveListenKey(e.listenKey)
			case <-e.listenKeyStop:
				return
			}
		}
	}()

	url := fmt.Sprintf("%s/%s", e.Option.WsHost, e.listenKey)
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return e.listenKey, err
	}
	conn.Subscribe(sub)
	return e.listenKey, nil
}

func (e *BinanceFutureWs) send(conn *exchanges.Connection, data Stream) (err error) {
	if err = conn.SendJsonMessage(data); err != nil {
		return err
	}
	return nil
}

func (e *BinanceFutureWs) messageHandler(url string, message []byte) {
	res := ResponseEvent{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] messageHandler unmarshal error:%v", err))
		return
	}
	switch res.Event {
	case "depthUpdate":
		if e.isIncrementalDepth == true {
			e.handleIncrementalDepth(url, message)
		} else {
			e.handleDepth(url, message)
		}

	case "24hrTicker":
		e.handleTicker(url, message)

	case "aggTrade":
		e.handleTrade(url, message)

	case "kline":
		e.handleKLine(url, message)

	case "ORDER_TRADE_UPDATE":
		e.handleOrder(url, message)

	case "ACCOUNT_UPDATE":
		e.handleBalance(url, message)
	case "markPriceUpdate":
		e.handleMarkPrice(url, message)
	}

}

func (e *BinanceFutureWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
	e.isSubUserData = false
}

func (e *BinanceFutureWs) disConnectedHandler(url string, err error) {
	// clear cache data, Prevent getting dirty data
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		delete(e.orderBooks, url)
		e.partialOrderBook = OrderBook{}
	})
}

func (e *BinanceFutureWs) closeHandler(url string) {
	// clear cache data and the connection
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
		if strings.Contains(url, e.listenKey) {
			e.listenKey = ""
			close(e.listenKeyStop)
		}
	})
	e.isSubUserData = false
}

func (e *BinanceFutureWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *BinanceFutureWs) heartbeatHandler(url string) {
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	conn.SendPongMessage([]byte("pong"))
}

func (e *BinanceFutureWs) handleDepth(url string, message []byte) {
	data := RawOrderBook{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleDepth - message Unmarshal to RawOrderBook error:%v", err))
		return
	}
	if data.LastUpdateID < e.partialOrderBook.LastUpdateID {
		return
	}

	e.partialOrderBook.Bids = wsex.Depth{}
	e.partialOrderBook.Asks = wsex.Depth{}
	e.partialOrderBook.update(data)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: e.partialOrderBook.OrderBook})
}

func (e *BinanceFutureWs) handleIncrementalDepth(url string, message []byte) {
	rawOB := RawOrderBook{}
	if err := json.Unmarshal(message, &rawOB); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleIncrementalDepth - message Unmarshal to RawOrderBook error:%v", err))
		return
	}
	market, err := e.GetMarketByID(rawOB.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}

	symbolOrderBook, ok := e.orderBooks[url]
	if !ok || symbolOrderBook == nil {
		_ = e.getSnapshotOrderBook(url, market, &SymbolOrderBook{})
		return
	}
	fullOrderBook, ok := (*symbolOrderBook)[market.Symbol]
	if !ok {
		_ = e.getSnapshotOrderBook(url, market, symbolOrderBook)
		return
	}
	if fullOrderBook != nil {
		first := rawOB.FirstUpdateID <= fullOrderBook.LastUpdateID+1 && rawOB.LastUpdateID >= fullOrderBook.LastUpdateID
		if first || rawOB.PreUpdateID == fullOrderBook.LastUpdateID {
			fullOrderBook.update(rawOB)
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: fullOrderBook.OrderBook})
		} else if rawOB.LastUpdateID < fullOrderBook.LastUpdateID {
			log.Printf("[BinanceWs] handleIncrementalDepth - recv old update data\n")
		} else {
			delete(*symbolOrderBook, market.Symbol)
			err := wsex.ExError{Code: wsex.ErrInvalidDepth,
				Message: fmt.Sprintf("[BinanceWs] handleIncrementalDepth - recv dirty data, new.FirstUpdateID: %v != old.LastUpdateID: %v ", rawOB.FirstUpdateID, fullOrderBook.LastUpdateID+1),
				Data:    map[string]interface{}{"symbol": fullOrderBook.Symbol}}
			e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: err})
		}
	}
}

func (e *BinanceFutureWs) handleTicker(url string, message []byte) {
	data := Ticker{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}
	ticker := data.parseTicker(market.Symbol)

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: ticker})
}

func (e *BinanceFutureWs) handleTrade(url string, message []byte) {
	data := Trade{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleTrade - message Unmarshal to trade error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}
	trade := data.parseTrade(market.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: trade})
}

func (e *BinanceFutureWs) handleKLine(url string, message []byte) {
	data := KLine{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleKLine - message Unmarshal to kline error:%v", err))
		return
	}
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}
	kline := wsex.KLine{
		Symbol:    market.Symbol,
		Timestamp: time.Duration(data.Line.BTimestamp),
		Open:      utils.SafeParseFloat(data.Line.Open),
		Close:     utils.SafeParseFloat(data.Line.Close),
		High:      utils.SafeParseFloat(data.Line.High),
		Low:       utils.SafeParseFloat(data.Line.Low),
		Volume:    utils.SafeParseFloat(data.Line.Volume),
	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgKLine, Data: kline})
}

func (e *BinanceFutureWs) handleMarkPrice(url string, message []byte) {
	data := MarkFundingRate{}
	restJson := jsoniter.Config{TagKey: "ws"}.Froze()
	err := restJson.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleMarkPrice - message Unmarshal to MarkPrice error:%v", err))
		return
	}
	market, _ := e.GetMarketByID(data.Symbol)
	markPrice := data.parserMarkPrice(market.Symbol)
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgMarkPrice, Data: markPrice})

}

func (e *BinanceFutureWs) handleBalance(url string, message []byte) {
	data := WsBalances{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleBalance - message Unmarshal to balance error:%v", err))
		return
	}
	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	balances.UpdateTime = time.Duration(data.Timestamp)
	Positions := make([]wsex.FuturePositons, 0)
	for _, b := range data.Event.WsB {
		balances.Balances[b.Currency] = b.parserWsBalance()
	}
	market, _ := e.GetMarketByID(data.Event.WsP[0].Symbol)
	for _, p := range data.Event.WsP {
		ps := p.parserWsPosition(market.Symbol)
		Positions = append(Positions, ps)
	}
	var futurePosition = wsex.FuturePositonsUpdate{
		Positons: make([]wsex.FuturePositons, 0),
	}
	futurePosition.Symbol = market.Symbol
	futurePosition.Positons = Positions
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgPositions, Data: futurePosition})
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}

func (e *BinanceFutureWs) handleOrder(url string, message []byte) {
	type WsOrders struct {
		Event         string `fj:"e"`
		EIgnore       int    `fj:"E"`
		Timestramp    int    `fj:"T"`
		FutureWsOrder Order  `fj:"o"`
	}
	var data WsOrders
	restJson := jsoniter.Config{TagKey: "fj"}.Froze()
	err := restJson.Unmarshal(message, &data)
	if err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleOrder - message Unmarshal to handleOrder error:%v", err))
		return
	}
	market, _ := e.GetMarketByID(data.FutureWsOrder.Symbol)
	order := data.FutureWsOrder.parseOrder(market.Symbol)
	order.Cost = fmt.Sprintf("%f", utils.SafeParseFloat(data.FutureWsOrder.AvePrice)*utils.SafeParseFloat(data.FutureWsOrder.Filled))
	order.CreateTime = time.Duration(data.Timestramp)

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
}

func (e *BinanceFutureWs) getSnapshotOrderBook(u string, market wsex.Market, symbolOrderBook *SymbolOrderBook) (err error) {
	time.Sleep(time.Second)

	var response struct {
		LastUpdateID int64         `json:"lastUpdateId"` // Last update ID
		Bids         wsex.RawDepth `json:"bids"`
		Asks         wsex.RawDepth `json:"asks"`
	}
	client := &http.Client{}
	reqUrl := fmt.Sprintf("%s/fapi/v1/depth?symbol=%s&limit=1000", e.Option.RestHost, market.SymbolID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return fmt.Errorf("[BinanceWs] getSnapshotOrderBook - request %s  error:%v", reqUrl, err)
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[BinanceWs] getSnapshotOrderBook - request %s  error:%v", reqUrl, err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("[BinanceWs] getSnapshotOrderBook - response body  error:%v", err)
	}
	json.Unmarshal(body, &response)
	if response.LastUpdateID == 0 {
		return fmt.Errorf("[BinanceWs] getSnapshotOrderBook - request url %s no data", reqUrl)
	}
	var orderBook OrderBook
	orderBook.Bids = orderBook.Bids.Update(response.Bids, true)
	orderBook.Asks = orderBook.Asks.Update(response.Asks, false)
	orderBook.LastUpdateID = response.LastUpdateID
	orderBook.Symbol = market.Symbol
	(*symbolOrderBook)[market.Symbol] = &orderBook
	e.orderBooks[u] = symbolOrderBook
	return
}

func (e *BinanceFutureWs) createListenKey() (string, error) {
	url := fmt.Sprintf("%s/fapi/v1/listenKey", e.Option.RestHost)
	type Listen struct {
		ListenKey string `json:"listenKey"`
	}
	var res Listen
	client := resty.New()
	if e.Option.ProxyUrl != "" {
		client.SetProxy(e.Option.ProxyUrl)
	}
	_, err := client.R().
		SetResult(&res).
		SetHeaders(map[string]string{
			"X-MBX-APIKEY": e.Option.AccessKey,
			"Content-Type": "application/x-www-form-urlencoded",
		}).
		Post(url)
	if err != nil {
		return "", fmt.Errorf("[BinanceFutureWs] createListenKey - request error:%v", err)
	}
	if res.ListenKey == "" {
		return "", errors.New("not found listenKey")
	}
	return res.ListenKey, nil
}

func (e *BinanceFutureWs) keepAliveListenKey(listenKey string) error {
	path := fmt.Sprintf("%s/fapi/v1/listenKey", e.Option.RestHost)
	body := fmt.Sprintf("listenKey=%s", listenKey)
	var uError UserDataStreamError
	client := resty.New()
	if e.Option.ProxyUrl != "" {
		client.SetProxy(e.Option.ProxyUrl)
	}
	_, err := client.R().
		SetHeaders(map[string]string{
			"X-MBX-APIKEY": e.Option.AccessKey,
			"Content-Type": "application/x-www-form-urlencoded",
		}).
		SetBody(body).
		SetError(&uError).
		Put(path)
	if err != nil {
		return fmt.Errorf("[BinanceWs] keepAliveListenKey - request error:%v", err)
	}
	if uError.Code < 0 {
		return uError
	}
	return nil
}

func (e *BinanceFutureWs) deleteListenKey(listenKey string) error {
	path := fmt.Sprintf("%s/fapi/v1/listenKey", e.Option.RestHost)
	body := fmt.Sprintf("listenKey=%s", listenKey)
	var uError UserDataStreamError
	client := resty.New()
	if e.Option.ProxyUrl != "" {
		client.SetProxy(e.Option.ProxyUrl)
	}
	_, err := client.R().
		SetHeaders(map[string]string{
			"X-MBX-APIKEY": e.Option.AccessKey,
			"Content-Type": "application/x-www-form-urlencoded",
		}).
		SetBody(body).
		SetError(&uError).
		Delete(path)
	if err != nil {
		return fmt.Errorf("[BinanceWs] deleteListenKey - request error:%v", err)
	}
	if uError.Code < 0 {
		return uError
	}
	return nil
}
