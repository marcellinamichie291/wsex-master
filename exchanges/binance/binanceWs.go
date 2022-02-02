/*
@Time : 2021/2/24 10:23 上午
@Author : shiguantian
@File : binance
@Software: GoLand
*/
package binance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/exchanges/websocket"
	. "github.com/shiguantian/wsex/utils"

	"errors"
)

type Stream struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int      `json:"id"`
}

func SubscribeStream(event ...string) Stream {
	return Stream{
		Method: "SUBSCRIBE",
		Params: append([]string{}, event...),
		Id:     time.Now().Nanosecond(),
	}
}

func UnSubscribeStream(event ...string) Stream {
	return Stream{
		Method: "UNSUBSCRIBE",
		Params: append([]string{}, event...),
		Id:     time.Now().Nanosecond(),
	}
}

type UserDataStreamError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (u UserDataStreamError) Error() string {
	return fmt.Sprintf("UserDataStreamError error, code:%v msg:%v", u.Code, u.Msg)
}

//The websocket function of binance is not well designed.
//first, there's no topic field passed by subscribe action in the async return data. so, it is impossible to directly distinguish whose data is
//Second, some return data don't even have event field, only determine what kind of data it is by parsing the string

type BinanceWs struct {
	exchanges.BaseExchange
	orderBooks       map[string]*SymbolOrderBook // orderbook's local cache of one symbol
	partialOrderBook OrderBook                   // Partial Book Depth
	errors           map[int]wsex.ExError
	listenKey        string // listenKey for User Data Streams, including account update,balance update,order update
	listenKeyStop    chan struct{}
}

func (e *BinanceWs) Init(option wsex.Options) {
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
		e.Option.WsHost = "wss://stream.binance.com:9443/ws"
	}
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.binance.com/api/v3"
	}
	e.listenKeyStop = make(chan struct{})
}

func (e *BinanceWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	e.RwLock.Lock()
	if !isIncremental {
		if e.partialOrderBook.Symbol != "" {
			//It's a poor design of binance, because there's no event field for this kind of return, it is impossible to distinguish whose data it is.
			//so only support one symbol data subscribe
			return "", errors.New("binance instance can only obtain one symbol partial order book at the same time")
		}
		e.partialOrderBook.Symbol = symbol
	}
	e.RwLock.Unlock()
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

func (e *BinanceWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "trade")
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	topic, err := e.getTopicBySymbol(symbol, "ticker")
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	topic := "!ticker@arr"
	topic = strings.ToLower(topic)
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	kt := parseKLienType(t)
	topic, err := e.getTopicBySymbol(symbol, fmt.Sprintf("kline_%s", kt))
	if topic == "" {
		return topic, err
	}
	return e.subscribe(e.Option.WsHost, topic, sub)
}

func (e *BinanceWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribeUserData(sub)
}

func (e *BinanceWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribeUserData(sub)
}

func (e *BinanceWs) UnSubscribe(event string, sub wsex.MessageChan) error {
	conn, err := e.ConnectionMgr.GetConnection(e.Option.WsHost, nil)
	if err != nil {
		return err
	}
	if err := e.send(conn, UnSubscribeStream(event)); err != nil {
		return err
	}

	conn.UnSubscribe(sub)
	return nil
}

func (e *BinanceWs) Connect(url string) (*exchanges.Connection, error) {
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

func (e *BinanceWs) subscribe(url, topic string, sub wsex.MessageChan) (string, error) {
	conn, err := e.ConnectionMgr.GetConnection(url, e.Connect)
	if err != nil {
		return "", err
	}

	if err := e.send(conn, SubscribeStream(topic)); err != nil {
		return "", err
	}
	conn.Subscribe(sub)
	return topic, nil
}

func (e *BinanceWs) subscribeUserData(sub wsex.MessageChan) (string, error) {
	e.RwLock.Lock()
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

func (e *BinanceWs) getTopicBySymbol(symbol, suffix string) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	topic := fmt.Sprintf("%s%s@%s", market.BaseID, market.QuoteID, suffix)
	return strings.ToLower(topic), nil
}

func (e *BinanceWs) send(conn *exchanges.Connection, data Stream) (err error) {
	if err = conn.SendJsonMessage(data); err != nil {
		return err
	}

	return nil
}

func (e *BinanceWs) messageHandler(url string, message []byte) {
	res := ResponseEvent{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] messageHandler unmarshal error:%v", err))
		return
	}

	switch res.Event {
	case "depthUpdate":
		//Diff. Depth Stream(Order book price and quantity depth updates used to locally manage an order book.)
		e.handleIncrementalDepth(url, message)
	case "24hrTicker":
		//Individual Symbol Ticker Streams(24hr rolling window ticker statistics for a single symbol)
		e.handleTicker(url, message)
	case "trade":
		//Trade Streams(The Trade Streams push raw trade information; each trade has a unique buyer and seller.)
		e.handleTrade(url, message)
	case "kline":
		//Kline/Candlestick Streams(The Kline/Candlestick Stream push updates to the current klines/candlestick every second.)
		e.handleKLine(url, message)
	case "executionReport":
		//Orders are updated with the executionReport event.
		//NEW - The order has been accepted into the engine.
		//CANCELED - The order has been canceled by the user.
		//REPLACED (currently unused)
		//REJECTED - The order has been rejected and was not processed. (This is never pushed into the User Data Stream)
		//TRADE - Part of the order or all of the order's quantity has filled.
		//EXPIRED - The order was canceled according to the order type's rules (e.g. LIMIT FOK orders with no fill, LIMIT IOC or MARKET orders that partially fill) or by the exchange, (e.g. orders canceled during liquidation, orders canceled during maintenance)
		e.handleOrder(url, message)
	case "outboundAccountPosition":
		//outboundAccountPosition is sent any time an account balance has changed and contains the assets that were possibly changed by the event that generated the balance change.
		e.handleBalance(url, false, message)
	case "balanceUpdate":
		//Balance Update occurs during the following:
		//Deposits or withdrawals from the account
		//Transfer of funds between accounts (e.g. Spot to Margin)
		e.handleBalance(url, true, message)
	default:
		//It's a poor design of binance, because there's no event field for this kind of return, it is impossible to distinguish whose data it is.
		//So the data will be published to all subscribers who have the same event type, and the user need distinguish the right market data by himself
		if bytes.Contains(message, []byte("lastUpdateId")) && bytes.Contains(message, []byte("bids")) {
			//Partial Book Depth Streams(Top bids and asks of specified level)
			e.handleDepth(url, message)
		}
	}
}

func (e *BinanceWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
}

func (e *BinanceWs) disConnectedHandler(url string, err error) {
	// clear cache data, Prevent getting dirty data
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		delete(e.orderBooks, url)
		e.partialOrderBook = OrderBook{}
	})
}

func (e *BinanceWs) closeHandler(url string) {
	// clear cache data and the connection
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
		if strings.Contains(url, e.listenKey) {
			e.listenKey = ""
			close(e.listenKeyStop)
		}
	})
}

func (e *BinanceWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *BinanceWs) heartbeatHandler(url string) {
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	conn.SendPongMessage([]byte("pong"))
}

func (e *BinanceWs) handleDepth(url string, message []byte) {
	data := RawOrderBook{}
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	if err := restJson.Unmarshal(message, &data); err != nil {
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

func (e *BinanceWs) handleIncrementalDepth(url string, message []byte) {
	rawOB := RawOrderBook{}
	if err := json.Unmarshal(message, &rawOB); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleIncrementalDepth - message Unmarshal to RawOrderBook error:%v", err))
		return
	}

	/*
		How to manage a local order book correctly
		1. Open a stream to wss://stream.binance.com:9443/ws/bnbbtc@depth.
		2. Buffer the events you receive from the stream.
		3. Get a depth snapshot from https://api.binance.com/api/v3/depth?symbol=BNBBTC&limit=1000 .
		4. Drop any event where u is <= lastUpdateId in the snapshot.
		5. The first processed event should have U <= lastUpdateId+1 AND u >= lastUpdateId+1.
		6. While listening to the stream, each new event's U should be equal to the previous event's u+1. restart from step3
		7. The data in each event is the absolute quantity for a price level.
		8. If the quantity is 0, remove the price level.
		9. Receiving an event that removes a price level that is not in your local order book can happen and is normal.
	*/
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
		if first || rawOB.FirstUpdateID == fullOrderBook.LastUpdateID+1 {
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

func (e *BinanceWs) handleTicker(url string, message []byte) {
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

func (e *BinanceWs) handleTrade(url string, message []byte) {
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

func (e *BinanceWs) handleKLine(url string, message []byte) {
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
		Open:      SafeParseFloat(data.Line.Open),
		Close:     SafeParseFloat(data.Line.Close),
		High:      SafeParseFloat(data.Line.High),
		Low:       SafeParseFloat(data.Line.Low),
		Volume:    SafeParseFloat(data.Line.Volume),
	}

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgKLine, Data: kline})
}

func (e *BinanceWs) handleBalance(url string, balanceUpdate bool, message []byte) {
	data := Balances{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleBalance - message Unmarshal to balance error:%v", err))
		return
	}

	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	if balanceUpdate {

	} else {
		balances.UpdateTime = time.Duration(data.Timestamp)
		for _, b := range data.Balances {
			balances.Balances[b.Currency] = b.parseBalance()
		}
	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}

func (e *BinanceWs) handleOrder(url string, message []byte) {
	data := Order{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[BinanceWs] handleOrder - message Unmarshal to handleOrder error:%v", err))
		return
	}
	market, _ := e.GetMarketByID(data.Symbol)
	order := data.parseOrder(market.Symbol)

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
}

func (e *BinanceWs) getSnapshotOrderBook(url string, market wsex.Market, symbolOrderBook *SymbolOrderBook) (err error) {
	time.Sleep(time.Second)

	var response struct {
		LastUpdateID int64         `json:"lastUpdateId"` // Last update ID
		Bids         wsex.RawDepth `json:"bids"`
		Asks         wsex.RawDepth `json:"asks"`
	}
	client := resty.New()
	reqUrl := fmt.Sprintf("%s/depth?symbol=%s&limit=1000", e.Option.RestHost, market.SymbolID)
	_, err = client.R().SetResult(&response).Get(reqUrl)
	if err != nil {
		return fmt.Errorf("[BinanceWs] getSnapshotOrderBook - request url %s error:%v", reqUrl, err)
	}
	if response.LastUpdateID == 0 {
		return fmt.Errorf("[BinanceWs] getSnapshotOrderBook - request url %s no data", reqUrl)
	}
	var orderBook OrderBook
	orderBook.Bids = orderBook.Bids.Update(response.Bids, true)
	orderBook.Asks = orderBook.Asks.Update(response.Asks, false)
	orderBook.LastUpdateID = response.LastUpdateID
	orderBook.Symbol = market.Symbol
	(*symbolOrderBook)[market.Symbol] = &orderBook
	e.orderBooks[url] = symbolOrderBook
	return
}

func (e *BinanceWs) createListenKey() (string, error) {
	url := fmt.Sprintf("%s/userDataStream", e.Option.RestHost)
	res := map[string]string{}
	client := resty.New()
	_, err := client.R().
		SetResult(&res).
		SetHeaders(map[string]string{
			"X-MBX-APIKEY": e.Option.AccessKey,
			"Content-Type": "application/x-www-form-urlencoded",
		}).
		Post(url)
	if err != nil {
		return "", fmt.Errorf("[BinanceWs] createListenKey - request error:%v", err)
	}

	listenKey, ok := res["listenKey"]
	if !ok {
		return "", errors.New("not found listenKey")
	}
	return listenKey, nil
}

func (e *BinanceWs) keepAliveListenKey(listenKey string) error {
	path := fmt.Sprintf("%s/userDataStream", e.Option.RestHost)
	body := fmt.Sprintf("listenKey=%s", listenKey)
	var uError UserDataStreamError
	client := resty.New()
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

func (e *BinanceWs) deleteListenKey(listenKey string) error {
	path := fmt.Sprintf("%s/userDataStream", e.Option.RestHost)
	body := fmt.Sprintf("listenKey=%s", listenKey)
	var uError UserDataStreamError
	client := resty.New()
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
