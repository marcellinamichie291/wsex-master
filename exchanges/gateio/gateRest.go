package gateio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/utils"
)

type GateRest struct {
	exchanges.BaseExchange
	errors map[string]int
}

func (e *GateRest) Init(options wsex.Options) {
	e.Option = options
	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.gateio.ws"
	}
	e.errors = map[string]int{
		"BALANCE_NOT_ENOUGH":                          wsex.ErrInsufficientFunds,
		"insufficient-balance":                        wsex.ErrInsufficientFunds,
		"insufficient-exchange-fund":                  wsex.ErrInsufficientFunds,
		"account-balance-insufficient-error":          wsex.ErrInsufficientFunds,
		"account-transfer-balance-insufficient_error": wsex.ErrInsufficientFunds,
		"ORDER_NOT_FOUND":                             wsex.ErrOrderNotFound,
		"not-found":                                   wsex.ErrOrderNotFound,
		"error":                                       wsex.ErrExchangeSystem,
	}
}

func (e *GateRest) FetchOrderBook(symbol string, size int) (OrderBook wsex.OrderBook, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	if size > 0 {
		if size > 100 {
			size = 100
		}
		params.Set("limit", strconv.Itoa(size))
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/spot/order_book", params, http.Header{})
	if err != nil {
		return
	}
	var data RawOrderBook
	err = json.Unmarshal(res, &data)
	if err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	OrderBook = data.parseOrderBook(symbol)
	return
}

func (e *GateRest) FetchTicker(symbol string) (ticker wsex.Ticker, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/spot/tickers", params, http.Header{})
	if err != nil {
		return
	}
	var data []Ticker
	err = json.Unmarshal(res, &data)
	if err != nil && len(data) == 0 {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	ticker = data[0].parseTicker(market.Symbol)
	return
}

func (e *GateRest) FetchAllTicker() (map[string]wsex.Ticker, error) {
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/spot/tickers", url.Values{}, http.Header{})
	if err != nil {
		return nil, err
	}
	var data []Ticker
	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Println(err)
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return nil, err
	}
	tickers := make(map[string]wsex.Ticker)
	for _, value := range data {
		market, err := e.GetMarketByID(value.Symbol)
		if err != nil {
			continue
		}
		tickers[market.Symbol] = value.parseTicker(market.Symbol)
	}
	return tickers, nil
}

func (e *GateRest) FetchTrade(symbol string) (trades []wsex.Trade, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	params.Set("limit", "10")
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/spot/trades", params, http.Header{})
	if err != nil {
		return
	}
	var data []Trade
	err = json.Unmarshal(res, &data)
	if err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	for _, t := range data {
		trades = append(trades, t.parseTrade(market.Symbol))
	}
	return
}

func (e *GateRest) FetchKLine(symbol string, t wsex.KLineType) (klines []wsex.KLine, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	params.Set("interval", parseKLienType(t))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/spot/candlesticks", params, http.Header{})
	if err != nil {
		return
	}
	var data [][]interface{}
	err = json.Unmarshal(res, &data)
	if err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	for _, k := range data {
		if len(k) < 6 {
			continue
		}
		var (
			timestamp,
			open, last, high, low, vol string
		)
		utils.SafeAssign(k[0], &timestamp)
		utils.SafeAssign(k[1], &vol)
		utils.SafeAssign(k[2], &last)
		utils.SafeAssign(k[3], &high)
		utils.SafeAssign(k[4], &low)
		utils.SafeAssign(k[5], &open)
		klines = append([]wsex.KLine{wsex.KLine{
			Symbol:    market.Symbol,
			Timestamp: time.Duration(utils.SafeParseFloat(timestamp)),
			Type:      t,
			Open:      utils.SafeParseFloat(open),
			Close:     utils.SafeParseFloat(last),
			High:      utils.SafeParseFloat(high),
			Low:       utils.SafeParseFloat(low),
			Volume:    utils.SafeParseFloat(vol),
		}}, klines...)
	}
	return
}

func (e *GateRest) FetchMarkets() (map[string]wsex.Market, error) {
	if len(e.Option.Markets) > 0 {
		return e.Option.Markets, nil
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/spot/currency_pairs", url.Values{}, http.Header{})
	if err != nil {
		return e.Option.Markets, nil
	}
	var Info ExchangeInfo
	err = json.Unmarshal(res, &Info)
	if err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return e.Option.Markets, err
	}
	e.Option.Markets = make(map[string]wsex.Market)
	for _, value := range Info {
		market := wsex.Market{
			SymbolID:        value.Symbol,
			Symbol:          fmt.Sprintf("%s/%s", strings.ToUpper(value.Base), strings.ToUpper(value.Quote)),
			BaseID:          strings.ToUpper(value.Base),
			QuoteID:         strings.ToUpper(value.Quote),
			PricePrecision:  value.PricePrecision,
			AmountPrecision: value.AmountPrecision,
			Lot:             value.MinAmount,
		}
		e.Option.Markets[market.Symbol] = market
	}
	return e.Option.Markets, nil
}

func (e *GateRest) FetchBalance() (balances map[string]wsex.Balance, err error) {
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/spot/accounts", url.Values{}, http.Header{})
	if err != nil {
		return
	}
	var data []Balance
	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Println(err)
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	balances = make(map[string]wsex.Balance)
	for _, b := range data {
		balances[b.Currency] = b.parserBalance()
	}
	return
}

func (e *GateRest) CreateOrder(symbol string, price, amount float64, side wsex.Side, tradeType wsex.TradeType, orderType wsex.OrderType, useClientID bool) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	params.Set("amount", utils.Round(amount, market.AmountPrecision, false))
	if side == wsex.Sell {
		params.Set("side", "sell")
	} else {
		params.Set("side", "buy")
	}
	switch tradeType {
	case wsex.MARKET:
		params.Set("type", "market")
	default:
		params.Set("type", "limit")
		params.Set("price", utils.Round(price, market.PricePrecision, false))
	}
	switch orderType {
	case wsex.Normal:
		params.Set("time_in_force", "gtc")
	case wsex.IOC:
		params.Set("time_in_force", "ioc")
	case wsex.PostOnly:
		params.Set("time_in_force", "poc")
	}
	if useClientID {
		params.Set("text", utils.GenerateOrderClientId(e.Option.ClientOrderIDPrefix, 32))
	}
	res, err := e.Fetch(e, exchanges.Private, exchanges.POST, "/spot/orders", params, http.Header{})
	if err != nil {
		return
	}
	type response struct {
		ID  string `json:"id"`
		CID string `json:"text"`
	}
	var data response
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order.ID = data.ID
	order.ClientID = data.CID
	return
}

func (e *GateRest) CancelOrder(symbol, orderID string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	_, err = e.Fetch(e, exchanges.Private, exchanges.DELETE, fmt.Sprintf("/spot/orders/%s", orderID), params, http.Header{})

	return err
}

func (e *GateRest) CancelAllOrders(symbol string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	_, err = e.Fetch(e, exchanges.Private, exchanges.DELETE, "/spot/orders", params, http.Header{})

	return err
}

func (e *GateRest) FetchOrder(symbol, orderID string) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, fmt.Sprintf("/spot/orders/%s", orderID), params, http.Header{})
	if err != nil {
		return
	}
	var data Order
	err = json.Unmarshal(res, &data)
	if err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order = data.parserOrder(market.Symbol)
	return
}

func (e *GateRest) FetchOpenOrders(symbol string, pageIndex, pageSize int) (orders []wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("currency_pair", market.SymbolID)
	params.Set("status", "open")
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/spot/orders", params, http.Header{})
	if err != nil {
		return
	}
	var data []Order
	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Println(err)
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	for _, d := range data {
		orders = append(orders, d.parserOrder(market.Symbol))
	}
	return
}

func (e *GateRest) Sign(access, method, function string, param url.Values, header http.Header) (request exchanges.Request) {
	request.Method = method
	request.Headers = header
	request.Headers.Set("Accept", "application/json")
	request.Headers.Set("Content-Type", "application/json")
	if access == exchanges.Public {
		request.Url = e.Option.RestHost + "/api/v4" + function
		if len(param) > 0 {
			request.Url = request.Url + "?" + param.Encode()
		}
	} else {
		t := time.Now().Unix()
		var plainText string
		if method == exchanges.GET || method == exchanges.DELETE {
			request.Url = e.Option.RestHost + "/api/v4" + function + "?" + param.Encode()
			HexBody, err := utils.HashSign(utils.SHA512, "", false)
			if err != nil {
				return
			}
			plainText = fmt.Sprintf("%s\n/api/v4%s\n%s\n%s\n%d", method, function, param.Encode(), HexBody, t)
		} else {
			request.Url = e.Option.RestHost + "/api/v4" + function
			data := make(map[string]string)
			for key, v := range param {
				data[key] = v[0]
			}
			dataStr, _ := json.Marshal(data)
			request.Body = string(dataStr)
			HexBody, err := utils.HashSign(utils.SHA512, string(dataStr), false)
			if err != nil {
				return
			}
			plainText = fmt.Sprintf("%s\n/api/v4%s\n%s\n%s\n%d", method, function, "", HexBody, t)
		}

		SignStr, err := utils.HmacSign(utils.SHA512, plainText, e.Option.SecretKey, false)
		if err != nil {
			return
		}
		request.Headers.Set("KEY", e.Option.AccessKey)
		request.Headers.Set("Timestamp", fmt.Sprintf("%d", t))
		request.Headers.Set("SIGN", SignStr)
	}
	return request
}

func (e *GateRest) HandleError(request exchanges.Request, response []byte) error {
	type Result struct {
		Label   string `json:"label"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}

	if strings.Contains(string(response), "label") {
		var result Result
		if err := json.Unmarshal(response, &result); err != nil {
			return nil
		}
		if result.Label == "" {
			return nil
		}

		errCode, ok := e.errors[result.Label]
		if ok {
			return wsex.ExError{Code: errCode, Message: result.Message}
		} else {
			return wsex.ExError{Code: wsex.UnHandleError, Message: fmt.Sprintf("code:%v msg:%v", result.Label, result.Message+result.Detail)}
		}
	}
	return nil
}
