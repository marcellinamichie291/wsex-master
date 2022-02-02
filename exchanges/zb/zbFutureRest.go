/*
@Time : 2021/5/8 10:48 上午
@Author : shiguantian
@File : zbRest
@Software: GoLand
*/
package zb

import (
	"encoding/json"
	"errors"
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

type ZbFutureRest struct {
	accountType  wsex.FutureAccountType
	contractType wsex.ContractType
	futuresKind  wsex.FuturesKind
	exchanges.BaseExchange
	errors map[int]int
}

func (e *ZbFutureRest) Init(option wsex.Options) {
	e.Option = option
	e.errors = map[int]int{
		10027: wsex.ErrExchangeSystem,
		10028: wsex.ErrExchangeSystem,
		10029: wsex.ErrExchangeSystem,
		11018: wsex.ErrInsufficientFunds,
		12000: wsex.ErrInvalidOrder,
		12001: wsex.ErrInvalidOrder,
		12002: wsex.ErrInvalidOrder,
		12003: wsex.ErrInvalidOrder,
		12004: wsex.ErrInvalidOrder,
		12005: wsex.ErrInvalidOrder,
		12006: wsex.ErrInvalidOrder,
		12007: wsex.ErrInvalidOrder,
		12008: wsex.ErrInvalidOrder,
		12009: wsex.ErrInvalidOrder,
		12012: wsex.ErrOrderNotFound,
	}

	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://futures.zb.com"
	}
	if e.Option.RestPrivateHost == "" {
		e.Option.RestPrivateHost = "https://futures.zb.com"
	}
}

func (e *ZbFutureRest) FetchOrderBook(symbol string, size int) (orderBook wsex.OrderBook, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("size", strconv.Itoa(size))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/depth", params, http.Header{})
	if err != nil {
		return
	}
	var response struct {
		Data struct {
			Asks [][]float64 `json:"asks"`
			Bids [][]float64 `json:"bids"`
		} `json:"data"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, ask := range response.Data.Asks {
		item := wsex.DepthItem{
			Price:  strconv.FormatFloat(ask[0], 'f', market.PricePrecision, 64),
			Amount: strconv.FormatFloat(ask[1], 'f', market.AmountPrecision, 64),
		}
		orderBook.Asks = append(orderBook.Asks, item)
	}
	for _, bid := range response.Data.Bids {
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

func (e *ZbFutureRest) FetchTicker(symbol string) (ticker wsex.Ticker, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/ticker", params, http.Header{})
	if err != nil {
		return
	}

	var response struct {
		Data map[string]FutureTicker `json:"data"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	if res, ok := response.Data[market.SymbolID]; ok {
		ticker = res.parseTicker()
		ticker.Symbol = symbol
	}
	return
}

func (e *ZbFutureRest) FetchAllTicker() (tickers map[string]wsex.Ticker, err error) {
	params := url.Values{}
	tickers = make(map[string]wsex.Ticker)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/ticker", params, http.Header{})
	if err != nil {
		return
	}

	var response struct {
		Data map[string]FutureTicker `json:"data"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	for s, t := range response.Data {
		symbol, err := e.GetMarketByID(s)
		if err == nil {
			ticker := t.parseTicker()
			ticker.Symbol = symbol.Symbol
			tickers[symbol.Symbol] = ticker
		}
	}
	return
}

func (e *ZbFutureRest) FetchTrade(symbol string) (trades []wsex.Trade, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/trade", params, http.Header{})
	if err != nil {
		return
	}

	var response struct {
		Data []FutureTrade `json:"data"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, t := range response.Data {
		trade := t.parseTrade()
		trade.Symbol = symbol
		trades = append(trades, trade)
	}
	return
}

func (e *ZbFutureRest) FetchKLine(symbol string, t wsex.KLineType) (klines []wsex.KLine, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("size", "100")
	switch t {
	case wsex.KLine1Week:
		params.Set("period", "1W")
	case wsex.KLine1Day:
		params.Set("period", "1D")
	case wsex.KLine3Day:
		params.Set("period", "3D")
	case wsex.KLine1Minute:
		params.Set("period", "1m")
	case wsex.KLine1Month:
		params.Set("period", "1M")
	case wsex.KLine1Hour:
		params.Set("period", "1H")
	case wsex.KLine4Hour:
		params.Set("period", "4H")
	case wsex.KLine12Hour:
		params.Set("period", "12H")
	case wsex.KLine6Hour:
		params.Set("period", "6H")
	case wsex.KLine2Hour:
		params.Set("period", "2H")
	default:
		err = errors.New("zb can not support kline interval")
		return
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/kline", params, http.Header{})
	if err != nil {
		return
	}
	var data = FutureKLine{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	klines = make([]wsex.KLine, 0)
	for _, ele := range data.Data {
		kline := wsex.KLine{
			Symbol:    market.Symbol,
			Timestamp: time.Duration(ele[5]),
			Type:      t,
			Open:      ele[0],
			Close:     ele[3],
			High:      ele[1],
			Low:       ele[2],
			Volume:    ele[4],
		}
		klines = append([]wsex.KLine{kline}, klines...)
	}
	return
}

func (e *ZbFutureRest) FetchMarkPrice(symbol string) (markPrice wsex.MarkPrice, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/markPrice", params, http.Header{})
	if err != nil {
		return
	}
	var response struct {
		Data map[string]string `json:"data"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	markPrice.Price = response.Data[market.SymbolID]
	markPrice.Symbol = symbol
	return
}

func (e *ZbFutureRest) FetchFundingRate(symbol string) (fundingRate wsex.FundingRate, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/public/v1/fundingRate", params, http.Header{})
	if err != nil {
		return
	}
	var response struct {
		Data FutureFundingRate `json:"data"`
	}
	if err = json.Unmarshal(res, &response); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	fundingRate = response.Data.parseFundingRate()
	return
}

func (e *ZbFutureRest) FetchMarkets() (map[string]wsex.Market, error) {
	if len(e.Option.Markets) > 0 {
		return e.Option.Markets, nil
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/Server/api/v2/config/marketList", url.Values{}, http.Header{})
	if err != nil {
		return e.Option.Markets, err
	}

	type Markets struct {
		Data []FutureMarket `json:"data"`
	}
	var markets = Markets{}
	if err = json.Unmarshal(res, &markets); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return e.Option.Markets, err
	}

	e.Option.Markets = make(map[string]wsex.Market)
	for _, value := range markets.Data {
		market := wsex.Market{
			SymbolID:        strings.ToUpper(value.Symbol),
			Symbol:          fmt.Sprintf("%v/%v", strings.ToUpper(value.BaseID), strings.ToUpper(value.QuoteID)),
			BaseID:          strings.ToUpper(value.BaseID),
			QuoteID:         strings.ToUpper(value.QuoteID),
			PricePrecision:  value.PricePrecision,
			AmountPrecision: value.AmountPrecision,
			Lot:             utils.SafeParseFloat(value.Lot),
		}
		e.Option.Markets[market.Symbol] = market
	}
	return e.Option.Markets, nil
}

func (e *ZbFutureRest) CreateOrder(symbol string, price, amount float64, side wsex.Side, tradeType wsex.TradeType, orderType wsex.OrderType, useClientID bool) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("price", utils.Round(price, market.PricePrecision, false))
	params.Set("amount", utils.Round(amount, market.AmountPrecision, false))
	t := "1"
	if side == wsex.OpenShort {
		t = "2"
	}
	if side == wsex.CloseLong {
		t = "3"
	}
	if side == wsex.CloseShort {
		t = "4"
	}
	params.Set("side", t)
	if tradeType == wsex.MARKET {
		params.Set("action", "11")
		return
	}
	params.Set("symbol", market.SymbolID)
	clientOrderId := ""
	if useClientID {
		clientOrderId = utils.GenerateOrderClientId(e.Option.ClientOrderIDPrefix, 32)
		params.Set("clientOrderId", clientOrderId)
	}
	res, err := e.Fetch(e, exchanges.Private, exchanges.POST, "/Server/api/v2/trade/order", params, http.Header{})
	if err != nil {
		return
	}

	type OrderID struct {
		ID string `json:"orderId"`
	}
	type response struct {
		ID OrderID `json:"data"`
	}
	data := response{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order.ID = data.ID.ID
	order.ClientID = clientOrderId
	return
}

func (e *ZbFutureRest) CancelOrder(symbol, orderID string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	if utils.IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("clientOrderId", orderID)
	} else {
		params.Set("orderId", orderID)
	}
	params.Set("symbol", market.SymbolID)
	_, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/Server/api/v2/trade/cancelOrder", params, http.Header{})
	return err
}

func (e *ZbFutureRest) CancelAllOrders(symbol string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	_, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/Server/api/v2/trade/cancelAllOrders", params, http.Header{})

	return err
}

func (e *ZbFutureRest) FetchOrder(symbol, orderID string) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	customerOrderId := ""
	if utils.IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("clientOrderId", orderID)
		customerOrderId = orderID
	} else {
		params.Set("orderId", orderID)
	}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/Server/api/v2/trade/getOrder", params, http.Header{})
	if err != nil {
		return
	}
	var data struct {
		Data FutureOrder `json:"data"`
	}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order = data.Data.parseOrder(market.Symbol)
	order.ClientID = customerOrderId
	return
}

func (e *ZbFutureRest) FetchOpenOrders(symbol string, pageIndex, pageSize int) (orders []wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("pageNum", strconv.Itoa(pageIndex))
	params.Set("pageSize", strconv.Itoa(pageSize))
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/Server/api/v2/trade/getUndoneOrders", params, http.Header{})
	if err != nil {
		return
	}
	orders = make([]wsex.Order, 0)
	var data struct {
		Data struct {
			List []FutureOrder
		} `json:"data"`
	}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	for _, order := range data.Data.List {
		orders = append(orders, order.parseOrder(market.Symbol))
	}
	return
}

func (e *ZbFutureRest) Setting(symbol string, leverage int, marginMode wsex.FutureMarginMode, positionMode wsex.FuturePositionsMode) error {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return err
	}
	if leverage > 100 {
		leverage = 100
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("leverage", strconv.Itoa(leverage))
	params.Set("futuresAccountType", e.getAccountType())
	_, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/Server/api/v2/setting/setLeverage", params, http.Header{})
	// if err != nil {
	// 	return err
	// }
	// params = url.Values{}
	// margin := 1
	// if marginMode == wsex.CrossedMargin {
	// 	margin = 2
	// }
	// params.Set("symbol", market.SymbolID)
	// params.Set("marginMode", strconv.Itoa(margin))
	// params.Set("futuresAccountType", e.getAccountType())
	// _, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/Server/api/v2/setting/setMarginMode", params, http.Header{})
	// if err != nil {
	// 	return err
	// }
	// params = url.Values{}
	// position := 1
	// if positionMode == wsex.TwoWay {
	// 	position = 2
	// }
	// params.Set("symbol", market.SymbolID)
	// params.Set("positionsMode", strconv.Itoa(position))
	// params.Set("futuresAccountType", e.getAccountType())
	// _, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/Server/api/v2/setting/setPositionsMode", params, http.Header{})
	return err
}

func (e *ZbFutureRest) FetchBalance() (balances map[string]wsex.Balance, err error) {
	params := url.Values{}
	params.Set("futuresAccountType", e.getAccountType())
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/Server/api/v2/Fund/balance", params, http.Header{})
	if err != nil {
		return
	}

	var data struct {
		Data []FutureBalance `json:"data"`
	}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	balances = make(map[string]wsex.Balance)
	for _, b := range data.Data {
		balance := b.parseBalance()
		balances[strings.ToUpper(balance.Asset)] = balance
	}
	return
}

func (e *ZbFutureRest) FetchPositions(symbol string) (positions []wsex.FuturePositons, err error) {
	params := url.Values{}
	if symbol != "" {
		market, err1 := e.GetMarket(symbol)
		if err1 != nil {
			err = err1
			return
		}
		params.Set("symbol", market.SymbolID)
	}
	params.Set("futuresAccountType", e.getAccountType())
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/Server/api/v2/Positions/getPositions", params, http.Header{})
	if err != nil {
		return
	}

	var data struct {
		Data []FuturePosition `json:"data"`
	}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	positions = make([]wsex.FuturePositons, 0)
	for _, position := range data.Data {
		m, err := e.GetMarketByID(position.Symbol)
		if err != nil {
			continue
		}
		positions = append(positions, position.parsePositions(m.BaseID, m.Symbol))
	}
	return
}

func (e *ZbFutureRest) FetchAccountInfo() (accountInfo wsex.FutureAccountInfo, err error) {
	params := url.Values{}
	params.Set("futuresAccountType", e.getAccountType())
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/Server/api/v2/Fund/getAccount", params, http.Header{})
	if err != nil {
		return
	}

	var data struct {
		Data FutureAccountInfo `json:"data"`
	}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	accountInfo = data.Data.parseAccountInfo()

	positions, err := e.FetchPositions("")
	if err != nil {
		return
	}
	accountInfo.Positions = make(map[string]map[wsex.PositionType]wsex.FuturePositons)
	for _, position := range positions {
		ps, ok := accountInfo.Positions[position.Coin]
		if !ok {
			ps = make(map[wsex.PositionType]wsex.FuturePositons)
		}
		ps[position.PositionType] = position
		accountInfo.Positions[position.Coin] = ps
	}
	return
}

func (e *ZbFutureRest) FetchAllPositions() (positions []wsex.FuturePositons, err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *ZbFutureRest) Sign(access, method, function string, param url.Values, header http.Header) exchanges.Request {
	var request = exchanges.Request{Method: method}
	if access == exchanges.Public {
		request.Url = fmt.Sprintf("%s%s%s", e.Option.RestHost, "", function)
		if len(param) > 0 {
			request.Url = request.Url + "?" + param.Encode()
		}
	} else {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		request.Headers = http.Header{}
		payload := timestamp + method + function + param.Encode()
		secretKey, err := utils.HashSign(utils.SHA1, e.Option.SecretKey, false)
		if err != nil {
			return request
		}
		signature, err := utils.HmacSign(utils.SHA256, payload, secretKey, true)
		if err != nil {
			return request
		}
		header.Set("ZB-APIKEY", e.Option.AccessKey)
		header.Set("ZB-TIMESTAMP", timestamp)
		header.Set("ZB-SIGN", signature)
		header.Set("ZB-LAN", "cn")
		if method == exchanges.POST {
			request.Body = utils.UrlValuesToJson(param)
			header.Set("Content-Type", "application/json")
			request.Url = fmt.Sprintf("%s%s", e.Option.RestPrivateHost, function)
		} else {
			// request.Headers = http.Header{}
			request.Url = fmt.Sprintf("%s%s", e.Option.RestPrivateHost, function) + "?" + param.Encode()
		}
	}
	return request
}

func (e *ZbFutureRest) getAccountType() string {
	accountType := 1
	if e.accountType == wsex.CoinMargin {
		accountType = 2
	}
	return strconv.Itoa(accountType)
}

func (e *ZbFutureRest) HandleError(request exchanges.Request, response []byte) error {
	type Result struct {
		Code    int
		Message string `json:"desc"`
	}
	var result Result
	if err := json.Unmarshal(response, &result); err != nil {
		return wsex.ExError{Message: string(response)}
	}

	if result.Code == 10000 {
		return nil
	}
	errCode, ok := e.errors[result.Code]
	if ok {
		return wsex.ExError{Code: errCode, Message: result.Message}
	} else {
		return wsex.ExError{Code: wsex.UnHandleError, Message: fmt.Sprintf("code:%v msg:%v", result.Code, result.Message)}
	}
}
