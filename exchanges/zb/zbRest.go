/*
@Time : 2021/5/8 10:48 上午
@Author : shiguantian
@File : zbRest
@Software: GoLand
*/
package zb

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shiguantian/wsex"

	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/utils"
)

type ZbRest struct {
	exchanges.BaseExchange
	errors map[int]int
}

func (e *ZbRest) Init(option wsex.Options) {
	e.Option = option
	e.errors = map[int]int{
		1001: wsex.ErrExchangeSystem,
		3001: wsex.ErrOrderNotFound,
		2001: wsex.ErrInsufficientFunds,
		2002: wsex.ErrInsufficientFunds,
		2003: wsex.ErrInsufficientFunds,
		2004: wsex.ErrInsufficientFunds,
		2005: wsex.ErrInsufficientFunds,
		2006: wsex.ErrInsufficientFunds,
		2007: wsex.ErrInsufficientFunds,
		2008: wsex.ErrInsufficientFunds,
		2009: wsex.ErrInsufficientFunds,
		4001: wsex.ErrDDoSProtection,
		4002: wsex.ErrDDoSProtection,
	}

	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.zb.com"
	}
	if e.Option.RestPrivateHost == "" {
		e.Option.RestPrivateHost = "https://trade.zb.com"
	}
}

func (e *ZbRest) FetchOrderBook(symbol string, size int) (orderBook wsex.OrderBook, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("market", market.SymbolID)
	params.Set("size", strconv.Itoa(size))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "depth", params, http.Header{})
	if err != nil {
		return
	}

	var response struct {
		Asks [][]float64 `json:"asks"`
		Bids [][]float64 `json:"bids"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, ask := range response.Asks {
		item := wsex.DepthItem{
			Price:  strconv.FormatFloat(ask[0], 'f', market.PricePrecision, 64),
			Amount: strconv.FormatFloat(ask[1], 'f', market.AmountPrecision, 64),
		}
		orderBook.Asks = append(orderBook.Asks, item)
	}
	for _, bid := range response.Bids {
		item := wsex.DepthItem{
			Price:  strconv.FormatFloat(bid[0], 'f', market.PricePrecision, 64),
			Amount: strconv.FormatFloat(bid[1], 'f', market.AmountPrecision, 64),
		}
		orderBook.Bids = append(orderBook.Bids, item)
	}
	sort.Reverse(orderBook.Bids)
	sort.Sort(orderBook.Asks)
	return
}

func (e *ZbRest) FetchTicker(symbol string) (ticker wsex.Ticker, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("market", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "ticker", params, http.Header{})
	if err != nil {
		return
	}

	var data TickerRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	ticker = data.parseTicker()
	return
}

func (e *ZbRest) FetchAllTicker() (tickers map[string]wsex.Ticker, err error) {
	params := url.Values{}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "allTicker", params, http.Header{})
	if err != nil {
		return
	}

	var data map[string]Ticker
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for s, t := range data {
		tickers[s] = t.parseTicker()
	}

	return
}

func (e *ZbRest) FetchTrade(symbol string) (trades []wsex.Trade, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("market", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "trades", params, http.Header{})
	if err != nil {
		return
	}

	var data []Trade
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, t := range data {
		trades = append(trades, t.parseTrade())
	}
	return
}

func (e *ZbRest) FetchKLine(symbol string, t wsex.KLineType) (klines []wsex.KLine, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	kLineType := ""
	switch t {
	case wsex.KLine1Minute:
		kLineType = "1min"
	case wsex.KLine3Minute:
		kLineType = "3min"
	case wsex.KLine5Minute:
		kLineType = "5min"
	case wsex.KLine15Minute:
		kLineType = "15min"
	case wsex.KLine30Minute:
		kLineType = "30min"
	case wsex.KLine1Hour:
		kLineType = "1hour"
	case wsex.KLine2Hour:
		kLineType = "2hour"
	case wsex.KLine4Hour:
		kLineType = "4hour"
	case wsex.KLine6Hour:
		kLineType = "6hour"
	case wsex.KLine12Hour:
		kLineType = "12hour"
	case wsex.KLine1Day:
		kLineType = "1day"
	case wsex.KLine3Day:
		kLineType = "3day"
	case wsex.KLine1Week:
		kLineType = "1week"
	}
	params.Set("market", market.SymbolID)
	params.Set("type", kLineType)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "kline", params, http.Header{})
	if err != nil {
		return
	}

	var data = KLine{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	klines = make([]wsex.KLine, 0)
	for _, ele := range data.Data {
		kline := wsex.KLine{
			Symbol:    market.Symbol,
			Timestamp: time.Duration(ele[0]),
			Type:      t,
			Open:      ele[1],
			Close:     ele[4],
			High:      ele[2],
			Low:       ele[3],
			Volume:    ele[5],
		}
		klines = append([]wsex.KLine{kline}, klines...)
	}
	return
}

func (e *ZbRest) FetchMarkets() (map[string]wsex.Market, error) {
	if len(e.Option.Markets) > 0 {
		return e.Option.Markets, nil
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "markets", url.Values{}, http.Header{})
	if err != nil {
		return e.Option.Markets, err
	}

	type Market struct {
		AmountScale int     `json:"amountScale"`
		PriceScale  int     `json:"priceScale"`
		MinAmount   float64 `json:"minAmount"`
		MinSize     float64 `json:"minSize"`
	}
	var markets map[string]Market = make(map[string]Market, 0)
	if err = json.Unmarshal(res, &markets); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return e.Option.Markets, err
	}

	e.Option.Markets = make(map[string]wsex.Market, 0)
	for key, value := range markets {
		coins := strings.Split(strings.ToUpper(key), "_")
		market := wsex.Market{
			SymbolID:        key,
			Symbol:          strings.Join(coins, "/"),
			BaseID:          coins[0],
			QuoteID:         coins[1],
			PricePrecision:  value.PriceScale,
			AmountPrecision: value.AmountScale,
			Lot:             value.MinAmount,
		}
		e.Option.Markets[market.Symbol] = market
	}
	return e.Option.Markets, nil
}

func (e *ZbRest) FetchBalance() (balances map[string]wsex.Balance, err error) {
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "getAccountInfo", url.Values{}, http.Header{})
	if err != nil {
		return
	}

	var data struct {
		Result struct {
			Balances []Balance `json:"coins"`
		} `json:"result"`
	}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	balances = make(map[string]wsex.Balance)
	for _, b := range data.Result.Balances {
		balance := b.parseBalance()
		balances[balance.Asset] = balance
	}
	return
}

func (e *ZbRest) CreateOrder(symbol string, price, amount float64, side wsex.Side, tradeType wsex.TradeType, orderType wsex.OrderType, useClientID bool) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("price", utils.Round(price, market.PricePrecision, false))
	params.Set("amount", utils.Round(amount, market.AmountPrecision, false))
	t := "1"
	if side == wsex.Sell {
		t = "0"
	}
	params.Set("tradeType", t)
	if orderType != wsex.Normal {
		ot := "1" // PostOnly
		if orderType == wsex.IOC {
			ot = "2"
		}
		params.Set("orderType", ot)
	}
	params.Set("currency", market.SymbolID)
	clientOrderId := ""
	if useClientID {
		clientOrderId = utils.GenerateOrderClientId(e.Option.ClientOrderIDPrefix, 32)
		params.Set("customerOrderId", clientOrderId)
	}
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "order", params, http.Header{})
	if err != nil {
		return
	}

	type response struct {
		ID string `json:"id"`
	}
	data := response{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order.ID = data.ID
	order.ClientID = clientOrderId
	return
}

func (e *ZbRest) CancelOrder(symbol, orderID string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	if utils.IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("customerOrderId", orderID)
	} else {
		params.Set("id", orderID)
	}
	params.Set("currency", market.SymbolID)
	_, err = e.Fetch(e, exchanges.Private, exchanges.GET, "cancelOrder", params, http.Header{})

	return err
}

func (e *ZbRest) CancelAllOrders(symbol string) (err error) {
	count := 0 //
	for count < 100 {
		orders, err := e.FetchOpenOrders(symbol, 1, 10)
		if err != nil || len(orders) == 0 {
			break
		}
		count++
		for _, order := range orders {
			_ = e.CancelOrder(symbol, order.ID)
			time.Sleep(time.Millisecond * 20)
		}
	}
	return
}

//FetchOrder : 获取订单详情
func (e *ZbRest) FetchOrder(symbol, orderID string) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	customerOrderId := ""
	if utils.IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("customerOrderId", orderID)
		customerOrderId = orderID
	} else {
		params.Set("id", orderID)
	}
	params.Set("currency", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "getOrder", params, http.Header{})
	if err != nil {
		return
	}

	var data OrderInfo
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	order = e.parseOrder(&data, market)
	order.ClientID = customerOrderId
	return
}

//FetchOpenOrders : 获取委托中的订单
func (e *ZbRest) FetchOpenOrders(symbol string, pageIndex, pageSize int) (orders []wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("pageIndex", strconv.Itoa(pageIndex))
	params.Set("pageSize", strconv.Itoa(pageSize))
	params.Set("currency", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "getUnfinishedOrdersIgnoreTradeType", params, http.Header{})
	if err != nil {
		return
	}
	var data = make([]OrderInfo, 0)
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	orders = make([]wsex.Order, len(data))
	for i, order := range data {
		orders[i] = e.parseOrder(&order, market)
	}
	return
}

func (e *ZbRest) Sign(access, method, function string, param url.Values, header http.Header) exchanges.Request {
	var request = exchanges.Request{Method: method}
	if access == exchanges.Public {
		request.Url = fmt.Sprintf("%s%s%s", e.Option.RestHost, "/data/v1/", function)
		if len(param) > 0 {
			request.Url = request.Url + "?" + param.Encode()
		}
	} else {
		param.Set("method", function)
		param.Set("accesskey", e.Option.AccessKey)
		auth := param.Encode()
		sha1 := sha1.New()
		sha1.Write([]byte(e.Option.SecretKey))
		secret := sha1.Sum([]byte(""))
		secretKey := fmt.Sprintf("%x", secret)

		hmac := hmac.New(md5.New, []byte(secretKey))
		hmac.Write([]byte(auth))
		result := hmac.Sum([]byte(""))
		signature := fmt.Sprintf("%x", result)
		signature = strings.ToLower(signature)
		suffix := fmt.Sprintf("sign=%v&reqTime=%v", signature, time.Now().Unix()*1000)
		request.Url = fmt.Sprintf("%s%s%s", e.Option.RestPrivateHost, "/api/", function) + "?" + auth + "&" + suffix
	}
	return request
}

func (e *ZbRest) HandleError(request exchanges.Request, response []byte) error {
	type Result struct {
		Code    int
		Message string
	}
	var result Result
	if err := json.Unmarshal(response, &result); err != nil {
		return nil
	}

	if result.Code == 0 || result.Code == 1000 {
		return nil
	}
	errCode, ok := e.errors[result.Code]
	if ok {
		return wsex.ExError{Code: errCode, Message: result.Message}
	} else {
		return wsex.ExError{Code: wsex.UnHandleError, Message: fmt.Sprintf("code:%v msg:%v", result.Code, result.Message)}
	}
	return nil
}

func (e *ZbRest) parseOrder(orderInfo *OrderInfo, market wsex.Market) (order wsex.Order) {
	order.Side = parseSide(orderInfo.Type)
	order.ID = orderInfo.ID
	order.Price = strconv.FormatFloat(orderInfo.Price, 'f', market.PricePrecision, 64)
	order.Amount = strconv.FormatFloat(orderInfo.TotalAmount, 'f', market.AmountPrecision, 64)
	order.Filled = fmt.Sprintf("%v", orderInfo.TradeAmount)
	order.Cost = fmt.Sprintf("%v", orderInfo.TradeMoney)
	order.CreateTime = time.Duration(orderInfo.TradeDate)
	order.Status = parseStatus(orderInfo.Status, orderInfo.TradeAmount)
	return
}
