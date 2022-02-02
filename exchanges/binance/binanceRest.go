/*
@Time : 2021/5/10 12:59 下午
@Author : shiguantian
@File : binanceRest
@Software: GoLand
*/
package binance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/utils"
	. "github.com/shiguantian/wsex/utils"
)

type RawError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type BinanceRest struct {
	exchanges.BaseExchange
	errors map[int]RawError
}

func (e *BinanceRest) Init(option wsex.Options) {
	e.Option = option
	e.errors = map[int]RawError{
		-2010: RawError{Code: wsex.ErrInsufficientFunds, Message: ""},
		20006: RawError{Code: wsex.ErrInsufficientFunds, Message: ""},
		-2013: RawError{Code: wsex.ErrOrderNotFound, Message: ""},
		-2011: RawError{Code: wsex.ErrOrderNotFound, Message: "Unknown order sent."},
	}

	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.binance.com"
	}
}

func (e *BinanceRest) FetchOrderBook(symbol string, size int) (orderBook wsex.OrderBook, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("limit", strconv.Itoa(size))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/v3/depth", params, http.Header{})
	if err != nil {
		return
	}

	var data RawOrderBook
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	if err = restJson.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	ob := OrderBook{}
	ob.update(data)

	return ob.OrderBook, nil
}

func (e *BinanceRest) FetchTicker(symbol string) (ticker wsex.Ticker, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/v3/ticker/24hr", params, http.Header{})
	if err != nil {
		return
	}

	var data Ticker
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	if err = restJson.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	ticker = data.parseTicker(market.Symbol)
	return
}

func (e *BinanceRest) FetchAllTicker() (tickers map[string]wsex.Ticker, err error) {
	params := url.Values{}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/v3/ticker/price", params, http.Header{})
	if err != nil {
		return
	}

	var data = make([]struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}, 0)
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	tickers = make(map[string]wsex.Ticker, 0)
	for _, t := range data {
		market, err := e.GetMarketByID(t.Symbol)
		if err != nil {
			continue
		}
		tickers[market.Symbol] = wsex.Ticker{Symbol: market.Symbol, Last: SafeParseFloat(t.Price)}
	}

	return
}

func (e *BinanceRest) FetchTrade(symbol string) (trades []wsex.Trade, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/v3/aggTrades", params, http.Header{})
	if err != nil {
		return
	}

	var data []Trade
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, t := range data {
		trades = append(trades, t.parseTrade(market.Symbol))
	}
	return
}

func (e *BinanceRest) FetchKLine(symbol string, t wsex.KLineType) (klines []wsex.KLine, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("interval", parseKLienType(t))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/v3/klines", params, http.Header{})
	if err != nil {
		return
	}

	var data [][]interface{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, ele := range data {
		if len(ele) < 6 {
			continue
		}
		var (
			timestamp                  float64
			open, last, high, low, vol string
		)
		SafeAssign(ele[0], &timestamp)
		SafeAssign(ele[1], &open)
		SafeAssign(ele[2], &high)
		SafeAssign(ele[3], &low)
		SafeAssign(ele[4], &last)
		SafeAssign(ele[5], &vol)
		kline := wsex.KLine{
			Symbol:    market.Symbol,
			Type:      t,
			Timestamp: time.Duration(timestamp),
			Open:      SafeParseFloat(open),
			High:      SafeParseFloat(high),
			Low:       SafeParseFloat(low),
			Close:     SafeParseFloat(last),
			Volume:    SafeParseFloat(vol),
		}
		klines = append([]wsex.KLine{kline}, klines...)
	}
	return
}

func (e *BinanceRest) FetchMarkets() (map[string]wsex.Market, error) {
	if len(e.Option.Markets) > 0 {
		return e.Option.Markets, nil
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/api/v3/exchangeInfo", url.Values{}, http.Header{})
	if err != nil {
		return e.Option.Markets, err
	}
	e.Option.Markets = make(map[string]wsex.Market)
	var info ExchangeInfo
	if err := json.Unmarshal(res, &info); err != nil {
		return e.Option.Markets, err
	}
	for _, m := range info.Markets {
		if m.Status != "TRADING" {
			continue
		}

		pricePrecision := m.QuotePrecision
		amountPrecision := m.BaseAssetPrecision
		for _, filter := range m.Filters {
			if filter.FilterType == "PRICE_FILTER" {
				re, _ := regexp.Compile(`0+$`)
				s := re.ReplaceAllString(filter.TickSize, "")
				pres := strings.Split(s, ".")
				if len(pres) == 2 {
					pricePrecision = len(pres[1])
				}
			}
			if filter.FilterType == "LOT_SIZE" {
				re, _ := regexp.Compile(`0+$`)
				s := re.ReplaceAllString(filter.StepSize, "")
				pres := strings.Split(s, ".")
				if len(pres) == 2 {
					amountPrecision = len(pres[1])
				}
			}
		}
		market := wsex.Market{
			SymbolID:        strings.ToUpper(m.Symbol),
			Symbol:          strings.ToUpper(fmt.Sprintf("%s/%s", m.BaseAsset, m.QuoteAsset)),
			BaseID:          strings.ToUpper(m.BaseAsset),
			QuoteID:         strings.ToUpper(m.QuoteAsset),
			PricePrecision:  pricePrecision,
			AmountPrecision: amountPrecision,
		}
		e.Option.Markets[market.Symbol] = market
	}
	return e.Option.Markets, nil
}

func (e *BinanceRest) FetchBalance() (balances map[string]wsex.Balance, err error) {
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/api/v3/account", url.Values{}, http.Header{})
	if err != nil {
		return
	}

	var data Balances
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	if err = restJson.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	balances = make(map[string]wsex.Balance)
	for _, b := range data.Balances {
		balance := b.parseBalance()
		balances[balance.Asset] = balance
	}
	return
}

func (e *BinanceRest) CreateOrder(symbol string, price, amount float64, side wsex.Side, tradeType wsex.TradeType, orderType wsex.OrderType, useClientID bool) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("quantity", utils.Round(amount, market.AmountPrecision, false))
	if side == wsex.Sell {
		params.Set("side", "SELL")
	} else if side == wsex.Buy {
		params.Set("side", "BUY")
	}
	switch tradeType {
	case wsex.MARKET:
		params.Set("type", "MARKET")
	default:
		params.Set("price", utils.Round(price, market.PricePrecision, false))
		params.Set("type", "LIMIT")
		params.Set("timeInForce", "GTC")
	}
	if useClientID {
		params.Set("newClientOrderId", GenerateOrderClientId(e.Option.ClientOrderIDPrefix, 32))
	}
	params.Set("newOrderRespType", "ACK")
	res, err := e.Fetch(e, exchanges.Private, exchanges.POST, "/api/v3/order", params, http.Header{})
	if err != nil {
		return
	}

	type response struct {
		ID  int64  `json:"orderId"`
		CID string `json:"clientOrderId"`
	}
	data := response{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order.ID = strconv.FormatInt(data.ID, 10)
	order.ClientID = data.CID
	return
}

func (e *BinanceRest) CancelOrder(symbol, orderID string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	if IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("origClientOrderId", orderID)
	} else {
		params.Set("orderId", orderID)
	}
	_, err = e.Fetch(e, exchanges.Private, exchanges.DELETE, "/api/v3/order", params, http.Header{})

	return err
}

func (e *BinanceRest) CancelAllOrders(symbol string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	_, err = e.Fetch(e, exchanges.Private, exchanges.DELETE, "/api/v3/openOrders", params, http.Header{})

	return err
}

//FetchOrder :
func (e *BinanceRest) FetchOrder(symbol, orderID string) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	if IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("origClientOrderId", orderID)
	} else {
		params.Set("orderId", orderID)
	}
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/api/v3/order", params, http.Header{})
	if err != nil {
		return
	}

	var data Order
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	if err = restJson.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	order = data.parseOrder(market.Symbol)
	return
}

//FetchOpenOrders :
func (e *BinanceRest) FetchOpenOrders(symbol string, pageIndex, pageSize int) (orders []wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/api/v3/openOrders", params, http.Header{})
	if err != nil {
		return
	}
	var data = make([]Order, 0)
	restJson := jsoniter.Config{TagKey: "rest"}.Froze()
	if err = restJson.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	orders = make([]wsex.Order, len(data))
	for i, order := range data {
		orders[i] = order.parseOrder(market.Symbol)
	}
	return
}

func (e *BinanceRest) Sign(access, method, function string, param url.Values, header http.Header) (request exchanges.Request) {
	request.Method = method
	request.Headers = header
	path := function
	if access == exchanges.Public {
		if len(param) > 0 {
			path = path + "?" + param.Encode()
		}
		request.Url = e.Option.RestHost + path
	} else {
		param.Set("recvWindow", "60000")
		tonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
		param.Set("timestamp", tonce)
		payload := param.Encode()
		signature, err := HmacSign(SHA256, payload, e.Option.SecretKey, false)
		if err != nil {
			return
		}
		param.Set("signature", signature)
		if method == exchanges.GET || method == exchanges.DELETE {
			path = path + "?" + param.Encode()
		} else {
			request.Body = param.Encode()
		}
		request.Headers.Set("X-MBX-APIKEY", e.Option.AccessKey)
		request.Url = e.Option.RestHost + path
	}
	return request
}

func (e *BinanceRest) HandleError(request exchanges.Request, response []byte) error {
	type Result struct {
		Code    int    `json:"code"`
		Message string `json:"msg"`
	}
	var result Result
	if err := json.Unmarshal(response, &result); err != nil {
		return nil
	}

	if result.Code == 0 || result.Code == 200 {
		return nil
	}
	rawErr, ok := e.errors[result.Code]
	if ok {
		if rawErr.Message == "" || strings.Contains(result.Message, rawErr.Message) {
			return wsex.ExError{Code: rawErr.Code, Message: result.Message}
		}
	}
	return wsex.ExError{Code: wsex.UnHandleError, Message: fmt.Sprintf("code:%v msg:%v", result.Code, result.Message)}
}
