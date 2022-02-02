/*
@Time : 2021/4/26 3:09 下午
@Author : shiguantian
@File : OkexWs
@Software: GoLand
*/
package okex

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/shiguantian/wsex/exchanges"

	"github.com/shiguantian/wsex"

	"github.com/shiguantian/wsex/exchanges/websocket"
	. "github.com/shiguantian/wsex/utils"

	"errors"
)

type OkexWs struct {
	exchanges.BaseExchange
	orderBooks map[string]*SymbolOrderBook
	errors     map[int]wsex.ExError
	loginLock  sync.Mutex
	loginChan  chan struct{}
	isLogin    bool
}

func (e *OkexWs) Init(option wsex.Options) {
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
	e.loginLock = sync.Mutex{}
	e.loginChan = make(chan struct{})
	e.isLogin = false
	if e.Option.WsHost == "" {
		e.Option.WsHost = "wss://real.okex.com:8443/ws/v3"
	}
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://www.okex.com"
	}
}

func (e *OkexWs) SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub wsex.MessageChan) (string, error) {
	return e.subscribe(e.Option.WsHost, "spot/depth_l2_tbt", symbol, false, sub)
}

func (e *OkexWs) SubscribeTrades(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribe(e.Option.WsHost, "spot/trade", symbol, false, sub)
}

func (e *OkexWs) SubscribeTicker(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribe(e.Option.WsHost, "spot/ticker", symbol, false, sub)
}

func (e *OkexWs) SubscribeAllTicker(sub wsex.MessageChan) (string, error) {
	return "", wsex.ExError{Code: wsex.NotImplement}
}

func (e *OkexWs) SubscribeKLine(symbol string, t wsex.KLineType, sub wsex.MessageChan) (string, error) {
	table := ""
	switch t {
	case wsex.KLine1Minute:
		table = "candle60s"
	case wsex.KLine3Minute:
		table = "candle180s"
	case wsex.KLine5Minute:
		table = "candle300s"
	case wsex.KLine15Minute:
		table = "candle900s"
	case wsex.KLine30Minute:
		table = "candle1800s"
	case wsex.KLine1Hour:
		table = "candle3600s"
	case wsex.KLine2Hour:
		table = "candle7200s"
	case wsex.KLine4Hour:
		table = "candle14400s"
	case wsex.KLine6Hour:
		table = "candle21600s"
	case wsex.KLine12Hour:
		table = "candle43200s"
	case wsex.KLine1Day:
		table = "candle86400s"
	case wsex.KLine1Week:
		table = "candle604800s"
	}
	return e.subscribe(e.Option.WsHost, fmt.Sprintf("spot/%s", table), symbol, false, sub)
}

func (e *OkexWs) SubscribeBalance(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribe(e.Option.WsHost, "spot/account", symbol, true, sub)
}

func (e *OkexWs) SubscribeOrder(symbol string, sub wsex.MessageChan) (string, error) {
	return e.subscribe(e.Option.WsHost, "spot/order", symbol, true, sub)
}

func (e *OkexWs) UnSubscribe(event string, sub wsex.MessageChan) error {
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

func (e *OkexWs) Connect(url string) (*exchanges.Connection, error) {
	conn := exchanges.NewConnection()
	err := conn.Connect(
		websocket.SetExchangeName("Okex"),
		websocket.SetWsUrl(url),
		websocket.SetProxyUrl(e.Option.ProxyUrl),
		websocket.SetIsAutoReconnect(e.Option.AutoReconnect),
		websocket.SetEnableCompression(false),
		websocket.SetHeartbeatIntervalTime(time.Second),
		websocket.SetReadDeadLineTime(time.Second*30),
		websocket.SetMessageHandler(e.messageHandler),
		websocket.SetErrorHandler(e.errorHandler),
		websocket.SetCloseHandler(e.closeHandler),
		websocket.SetReConnectedHandler(e.reConnectedHandler),
		websocket.SetDisConnectedHandler(e.disConnectedHandler),
		websocket.SetHeartbeatHandler(e.heartbeatHandler),
		websocket.SetDecompressHandler(e.decompressHandler),
	)

	return conn, err
}

func (e *OkexWs) subscribe(url, table, symbol string, needLogin bool, sub wsex.MessageChan) (string, error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return "", err
	}
	topic := fmt.Sprintf("%s:%s", table, market.SymbolID)
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

	if table == "spot/account" {
		e.send(conn, SubscribeStream(fmt.Sprintf("%s:%s", table, market.BaseID)))
		e.send(conn, SubscribeStream(fmt.Sprintf("%s:%s", table, market.QuoteID)))
	} else {
		if err := e.send(conn, SubscribeStream(topic)); err != nil {
			return "", err
		}
	}
	conn.Subscribe(sub)
	return topic, nil
}

func (e *OkexWs) send(conn *exchanges.Connection, data Stream) (err error) {
	if conn == nil {
		return errors.New("connect session is nil")
	}
	if err = conn.SendJsonMessage(data); err != nil {
		return err
	}

	return nil
}

func (e *OkexWs) messageHandler(url string, message []byte) {
	if string(message) == "pong" {
		return
	}

	res := ResponseEvent{}
	if err := json.Unmarshal(message, &res); err != nil {
		e.errorHandler(url, fmt.Errorf("[Okex] messageHandler unmarshal error:%v", err))
		return
	}

	if res.Event == "error" {
		e.errorHandler(url, fmt.Errorf("[OkexWs] messageHandler - business errcode:%v errmsg:%v", res.ErrorCode, res.Message))
		e.ConnectionMgr.Publish(url, wsex.ErrorMessage(e.handleError(res)))
		return
	} else if res.Event == "login" {
		e.isLogin = true
		e.loginChan <- struct{}{}
		return
	} else if res.Event != "" {
		log.Printf("[OkexWs] messageHandler - op:%v channel:%s success\n", res.Event, res.Channel)
		return
	}

	switch res.Table {
	case "spot/depth_l2_tbt", "spot/depth", "spot/depth5":
		e.handleDepth(url, message)
	case "spot/ticker":
		e.handleTicker(url, message)
	case "spot/trade":
		e.handleTrade(url, message)
	case "spot/candle60s", "spot/candle180s", "spot/candle300s", "spot/candle900s", "spot/candle1800s", "spot/candle3600s",
		"spot/candle7200s", "spot/candle14400s", "spot/candle21600s", "spot/candle43200s", "spot/candle86400s", "spot/candle604800s":
		e.handleKLine(url, message)
	case "spot/order":
		e.handleOrder(url, message)
	case "spot/account":
		e.handleBalance(url, message)
	default:
		e.errorHandler(url, fmt.Errorf("[OkexWs] messageHandler - not support this event type :%v", res.Table))
	}
}

func (e *OkexWs) reConnectedHandler(url string) {
	e.BaseExchange.ReConnectedHandler(url, nil)
}
func (e *OkexWs) disConnectedHandler(url string, err error) {
	e.BaseExchange.DisConnectedHandler(url, err, func() {
		e.isLogin = false
		delete(e.orderBooks, url)
	})
}

func (e *OkexWs) closeHandler(url string) {
	// clear cache data and the connection
	e.BaseExchange.CloseHandler(url, func() {
		delete(e.orderBooks, url)
	})
}

func (e *OkexWs) errorHandler(url string, err error) {
	e.BaseExchange.ErrorHandler(url, err, nil)
}

func (e *OkexWs) heartbeatHandler(url string) {
	//Log.Infof("[OkexWs] heartbeatHandler - %s", data)
	conn, err := e.ConnectionMgr.GetConnection(url, nil)
	if err != nil {
		return
	}
	conn.SendMessage([]byte("ping"))
}

func (e *OkexWs) decompressHandler(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (e *OkexWs) handleDepth(url string, message []byte) {
	rawOB := OrderBookRes{}
	if err := json.Unmarshal(message, &rawOB); err != nil {
		e.errorHandler(url, fmt.Errorf("[OkexWs] handleDepth - message Unmarshal to UpdateOrderBook error:%v", err))
		return
	}

	if len(rawOB.Data) < 1 {
		return
	}
	data := rawOB.Data[0]
	market, err := e.GetMarketByID(data.Symbol)
	if err != nil {
		e.errorHandler(url, err)
		return
	}

	symbolOrderBook, exit := e.orderBooks[url]
	newOrderBook := &OrderBook{}
	newOrderBook.Symbol = market.Symbol

	//The 400 entries of market depth data of the order book that return for the first time after subscription will be pushed;
	//subsequently as long as there's any change of market depth data of the order book, the changes will be pushed tick by tick.
	if rawOB.Action == "partial" {
		symbolOrderBook = &SymbolOrderBook{}
	} else if rawOB.Action == "update" {
		if !exit || symbolOrderBook == nil {
			return
		}
		cacheOrderBook, ok := (*symbolOrderBook)[market.Symbol]
		if ok {
			newOrderBook = cacheOrderBook
		}
	}
	newOrderBook.update(data)

	crc32BaseBuffer, expectCrc32 := e.calCrc32(&newOrderBook.Asks, &newOrderBook.Bids)
	if expectCrc32 == data.Checksum {
		(*symbolOrderBook)[market.Symbol] = newOrderBook
		e.orderBooks[url] = symbolOrderBook
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: newOrderBook.OrderBook})
	} else {
		err := wsex.ExError{Code: wsex.ErrInvalidDepth,
			Message: fmt.Sprintf("[OkexWs] handleDepth - recv dirty data, Checksum's not correct. LocalString: %s, LocalCrc32: %d, RemoteCrc32: %d",
				crc32BaseBuffer.String(), expectCrc32, data.Checksum),
			Data: map[string]interface{}{"symbol": newOrderBook.Symbol}}
		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrderBook, Data: err})
	}
}

func (e *OkexWs) handleTicker(url string, message []byte) {
	data := TickerRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[OkexWs] handleTicker - message Unmarshal to ticker error:%v", err))
		return
	}

	tickers := make([]wsex.Ticker, 0)
	for _, t := range data.Data {
		market, err := e.GetMarketByID(t.Symbol)
		if err != nil {
			e.errorHandler(url, err)
			continue
		}
		ticker := t.parseTicker(market.Symbol)
		tickers = append(tickers, ticker)
	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTicker, Data: tickers})
}

func (e *OkexWs) handleTrade(url string, message []byte) {
	data := TradeRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[OkexWs] handleTrade - message Unmarshal to trade error:%v", err))
		return
	}

	trades := make([]wsex.Trade, 0)
	for _, t := range data.Data {
		market, err := e.GetMarketByID(t.Symbol)
		if err != nil {
			e.errorHandler(url, err)
			continue
		}
		trade := t.parseTrade(market.Symbol)
		trades = append(trades, trade)
	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgTrade, Data: trades})
}

func (e *OkexWs) handleKLine(url string, message []byte) {
	data := KLineRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[OkexWs] handleKLine - message Unmarshal to KLine error:%v", err))
		return
	}

	klines := make([]wsex.KLine, 0)
	for _, k := range data.Data {
		market, err := e.GetMarketByID(k.Symbol)
		if err != nil {
			e.errorHandler(url, err)
			continue
		}
		kline := k.Candle.parseKLine(market.Symbol)
		klines = append(klines, kline)

	}
	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgKLine, Data: klines})
}

func (e *OkexWs) handleBalance(url string, message []byte) {
	data := BalanceRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[OkexWs] handleBalance - message Unmarshal to balance error:%v", err))
		return
	}

	balances := wsex.BalanceUpdate{Balances: make(map[string]wsex.Balance)}
	for _, b := range data.Data {
		balance := b.parseBalance()
		balances.Balances[balance.Asset] = balance
	}

	e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgBalance, Data: balances})
}

func (e *OkexWs) handleOrder(url string, message []byte) {
	data := OrderRes{}
	if err := json.Unmarshal(message, &data); err != nil {
		e.errorHandler(url, fmt.Errorf("[OkexWs] handleOrder - message Unmarshal to Order error:%v", err))
		return
	}

	for _, d := range data.Data {
		market, err := e.GetMarketByID(d.Symbol)
		if err != nil {
			e.errorHandler(url, err)
			continue
		}
		order := d.parseOrder(market.Symbol)

		e.ConnectionMgr.Publish(url, wsex.Message{Type: wsex.MsgOrder, Data: order})
	}
}

func (e *OkexWs) login(conn *exchanges.Connection) error {
	timestamp := EpochTime()

	preHash := e.preHashString(timestamp, "GET", "/users/self/verify", "")
	if sign, err := HmacSign(SHA256, preHash, e.Option.SecretKey, true); err != nil {
		return err
	} else {
		stream := LoginStream(e.Option.AccessKey, e.Option.PassPhrase, timestamp, sign)
		if err := e.send(conn, stream); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}

func (e *OkexWs) preHashString(timestamp string, method string, requestPath string, body string) string {
	return timestamp + strings.ToUpper(method) + requestPath + body
}

func (e *OkexWs) calCrc32(askDepths *wsex.Depth, bidDepths *wsex.Depth) (bytes.Buffer, int32) {
	crc32BaseBuffer := bytes.Buffer{}
	crcAskDepth, crcBidDepth := 25, 25
	if len(*askDepths) < 25 {
		crcAskDepth = len(*askDepths)
	}
	if len(*bidDepths) < 25 {
		crcBidDepth = len(*bidDepths)
	}
	if crcAskDepth == crcBidDepth {
		for i := 0; i < crcAskDepth; i++ {
			if crc32BaseBuffer.Len() > 0 {
				crc32BaseBuffer.WriteString(":")
			}
			crc32BaseBuffer.WriteString(
				fmt.Sprintf("%v:%v:%v:%v",
					(*bidDepths)[i].Price, (*bidDepths)[i].Amount,
					(*askDepths)[i].Price, (*askDepths)[i].Amount))
		}
	} else {
		for i := 0; i < crcBidDepth; i++ {
			if crc32BaseBuffer.Len() > 0 {
				crc32BaseBuffer.WriteString(":")
			}
			crc32BaseBuffer.WriteString(
				fmt.Sprintf("%v:%v", (*bidDepths)[i].Price, (*bidDepths)[i].Amount))
		}

		for i := 0; i < crcAskDepth; i++ {
			if crc32BaseBuffer.Len() > 0 {
				crc32BaseBuffer.WriteString(":")
			}
			crc32BaseBuffer.WriteString(
				fmt.Sprintf("%v:%v", (*askDepths)[i].Price, (*askDepths)[i].Amount))
		}
	}
	expectCrc32 := int32(crc32.ChecksumIEEE(crc32BaseBuffer.Bytes()))
	return crc32BaseBuffer, expectCrc32
}

func (e *OkexWs) handleError(res ResponseEvent) wsex.ExError {
	err, ok := e.errors[res.ErrorCode]
	if ok {
		return err
	}
	return wsex.ExError{Code: res.ErrorCode, Message: res.Message}
}
