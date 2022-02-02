package huobi

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

	jsoniter "github.com/json-iterator/go"
	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	"github.com/shiguantian/wsex/utils"
	. "github.com/shiguantian/wsex/utils"
)

type HuobiRest struct {
	exchanges.BaseExchange
	errors    map[string]int
	SymbolMap map[string]string
}

var AccountId int = 0

func (e *HuobiRest) Init(option wsex.Options) {
	e.Option = option

	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.huobi.pro"
	}
	if e.Option.RestPrivateHost == "" {
		e.Option.RestPrivateHost = "https://api.huobi.pro"
	}
	e.SymbolMap = make(map[string]string)
	e.errors = map[string]int{
		"order-accountbalance-error":                  wsex.ErrInsufficientFunds,
		"insufficient-balance":                        wsex.ErrInsufficientFunds,
		"insufficient-exchange-fund":                  wsex.ErrInsufficientFunds,
		"account-balance-insufficient-error":          wsex.ErrInsufficientFunds,
		"account-transfer-balance-insufficient_error": wsex.ErrInsufficientFunds,
		"base-not-found":                              wsex.ErrOrderNotFound,
		"not-found":                                   wsex.ErrOrderNotFound,
		"error":                                       wsex.ErrExchangeSystem,
	}
}

func (e *HuobiRest) FetchOrderBook(symbol string, size int) (orderBook wsex.OrderBook, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	params.Set("type", "step"+strconv.Itoa(size))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/market/depth", params, http.Header{})
	if err != nil {
		return
	}
	var orderBookRes OrderBookRes
	if err = json.Unmarshal(res, &orderBookRes); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	orderBook = orderBookRes.parseOrderBook(symbol)
	return
}

func (e *HuobiRest) FetchTicker(symbol string) (ticker wsex.Ticker, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/market/detail/merged", params, http.Header{})
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

func (e *HuobiRest) FetchAllTicker() (tickers map[string]wsex.Ticker, err error) {
	params := url.Values{}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/market/tickers", params, http.Header{})
	if err != nil {
		return
	}
	var data AllTickerRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	tickers = data.parseAllTickers(e.SymbolMap)
	return
}

func (e *HuobiRest) FetchTrade(symbol string) (trades []wsex.Trade, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/market/trade", params, http.Header{})
	if err != nil {
		return
	}
	var data TradeRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}

	for _, t := range data.Trade.Data {
		trades = append(trades, t.parseTrade())
	}
	return
}

func (e *HuobiRest) FetchKLine(symbol string, t wsex.KLineType) (klines []wsex.KLine, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	switch t {
	case wsex.KLine15Minute:
		params.Set("period", "15min")
	case wsex.KLine1Week:
		params.Set("period", "1week")
	case wsex.KLine1Day:
		params.Set("period", "1day")
	case wsex.KLine1Minute:
		params.Set("period", "1min")
	case wsex.KLine1Month:
		params.Set("period", "1mon")
	case wsex.KLine1Hour:
		params.Set("period", "60min")
	case wsex.KLine30Minute:
		params.Set("period", "30min")
	case wsex.KLine4Hour:
		params.Set("period", "4hour")
	case wsex.KLine5Minute:
		params.Set("period", "5min")
	default:
		return nil, errors.New("huobipro can not support kline interval")
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/market/history/kline", params, http.Header{})
	if err != nil {
		return
	}
	var data KLineRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	klines = data.parseKLine(market, t)
	return
}

func (e *HuobiRest) FetchMarkets() (map[string]wsex.Market, error) {
	if len(e.Option.Markets) > 0 {
		return e.Option.Markets, nil
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/v1/common/symbols", url.Values{}, http.Header{})
	if err != nil {
		return e.Option.Markets, err
	}

	var markets SymbolListRes
	if err = json.Unmarshal(res, &markets); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return e.Option.Markets, err
	}

	e.Option.Markets = make(map[string]wsex.Market)
	e.SymbolMap = make(map[string]string)
	for _, value := range markets.Data {
		market := wsex.Market{
			SymbolID:        value.Symbol,
			Symbol:          strings.ToUpper(fmt.Sprintf("%v/%v", value.Base, value.Quote)),
			BaseID:          strings.ToUpper(value.Base),
			QuoteID:         strings.ToUpper(value.Quote),
			PricePrecision:  value.PricePrecision,
			AmountPrecision: value.AmountPrecision,
			Lot:             value.MinAmount,
		}
		e.Option.Markets[market.Symbol] = market
		e.SymbolMap[value.Symbol] = market.Symbol
	}
	return e.Option.Markets, nil
}

func (e *HuobiRest) GetAccount() (accountId int, err error) {
	if AccountId == 0 {
		params := url.Values{}
		res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/v1/account/accounts", params, http.Header{})
		if err != nil {
			return 0, err
		}

		var accountData AccountRes
		if err = json.Unmarshal(res, &accountData); err != nil {
			err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
			return 0, err
		}
		AccountId = accountData.Data[0].Id
	}
	return AccountId, nil
}

func (e *HuobiRest) FetchBalance() (balances map[string]wsex.Balance, err error) {
	accountId, err := e.GetAccount()
	if err != nil {
		return
	}
	params := url.Values{}
	params.Add("account-id", strconv.Itoa(int(accountId)))
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, "/v1/account/accounts/"+strconv.Itoa(accountId)+"/balance", url.Values{}, http.Header{})
	if err != nil {
		return
	}
	var data BalanceRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	balances = data.parseBalance()
	return
}

func (e *HuobiRest) CreateOrder(symbol string, price, amount float64, side wsex.Side, tradeType wsex.TradeType, orderType wsex.OrderType, useClientID bool) (order wsex.Order, err error) {
	accountId, err := e.GetAccount()
	if err != nil {
		return
	}
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Add("account-id", strconv.Itoa(int(accountId)))
	params.Set("symbol", market.SymbolID)
	params.Set("amount", utils.Round(amount, market.AmountPrecision, false))
	if side == wsex.Sell {
		switch tradeType {
		case wsex.MARKET:
			params.Set("type", "sell-market")
			params.Set("amount", utils.Round(amount*price, market.AmountPrecision, false))
		default:
			params.Set("price", utils.Round(price, market.PricePrecision, false))
			params.Set("type", "sell-limit")
		}
	} else if side == wsex.Buy {
		switch tradeType {
		case wsex.MARKET:
			params.Set("type", "buy-market")
			params.Set("amount", utils.Round(amount*price, market.AmountPrecision, false))
		default:
			params.Set("price", utils.Round(price, market.PricePrecision, false))
			params.Set("type", "buy-limit")
		}
	}
	if useClientID {
		params.Set("client-order-id", GenerateOrderClientId(e.Option.ClientOrderIDPrefix, 32))
	}
	res, err := e.Fetch(e, exchanges.Private, exchanges.POST, "/v1/order/orders/place", params, http.Header{})
	if err != nil {
		return
	}
	type response struct {
		ID string `json:"data"`
	}
	data := response{}
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order.ID = data.ID
	order.ClientID = params.Get("client-order-id")
	return
}

func (e *HuobiRest) CancelOrder(symbol, orderID string) (err error) {
	params := url.Values{}
	if IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("client-order-id", orderID)
		_, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/v1/order/orders/submitCancelClientOrder", params, http.Header{})
		return err
	} else {
		params.Set("order-id", orderID)
		_, err = e.Fetch(e, exchanges.Private, exchanges.POST, "/v1/order/orders/"+orderID+"/submitcancel", params, http.Header{})
		return err
	}
}

func (e *HuobiRest) CancelAllOrders(symbol string) (err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("symbol", market.SymbolID)
	toDelete := true
	for toDelete {
		res, cancelErr := e.Fetch(e, exchanges.Private, exchanges.POST, "/v1/order/orders/batchCancelOpenOrders", params, http.Header{})
		if cancelErr != nil {
			return cancelErr
		}
		type CancelAllOrdersRes struct {
			Data struct {
				NextId int64 `json:"next-id"`
			} `json:"data"`
		}
		var data CancelAllOrdersRes
		if err = json.Unmarshal(res, &data); err != nil {
			err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
			return
		}
		toDelete = data.Data.NextId != -1
	}

	return err
}

func (e *HuobiRest) FetchOrder(symbol, orderID string) (order wsex.Order, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	var path string
	if IsClientOrderID(orderID, e.Option.ClientOrderIDPrefix) {
		params.Set("clientOrderId", orderID)
		path = "/v1/order/orders/getClientOrder"
	} else {
		params.Add("order-id", orderID)
		path = "/v1/order/orders/" + orderID
	}

	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, path, params, http.Header{})
	if err != nil {
		return
	}
	var data OrderRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	order = data.parseOrder(symbol, market)
	return
}

func (e *HuobiRest) FetchOpenOrders(symbol string, pageIndex, pageSize int) (orders []wsex.Order, err error) {
	accountId, err := e.GetAccount()
	if err != nil {
		return
	}
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("account-id", strconv.Itoa(int(accountId)))
	params.Set("symbol", market.SymbolID)
	function := "/v1/order/openOrders"
	res, err := e.Fetch(e, exchanges.Private, exchanges.GET, function, params, http.Header{})
	if err != nil {
		return
	}
	var data OpenOrderList
	openJson := jsoniter.Config{TagKey: "open"}.Froze()
	if err = openJson.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	orders = make([]wsex.Order, len(data.Data))
	for i, order := range data.Data {
		orders[i] = (&OrderRes{Data: order}).parseOrder(market.Symbol, market)
	}
	return
}

func (e *HuobiRest) Sign(access, method, function string, param url.Values, header http.Header) (request exchanges.Request) {
	request.Headers = header
	request.Method = method
	if access == exchanges.Public {
		request.Url = fmt.Sprintf("%s%s", e.Option.RestHost, function)
		if len(param) > 0 {
			request.Url = request.Url + "?" + param.Encode()
		}
	} else {
		payload := ""
		payload += "AccessKeyId=" + e.Option.AccessKey + "&SignatureMethod=HmacSHA256&SignatureVersion=2&Timestamp=" + url.QueryEscape(time.Now().UTC().Format("2006-01-02T15:04:05"))
		plainText := ""
		if method == exchanges.GET {
			plainText += "GET\n"
			keys := []string{}
			for key := range param {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				payload += "&" + key + "=" + url.QueryEscape(param.Get(key))
			}
		}
		if method == exchanges.POST {
			plainText += "POST\n"
			request.Body = UrlValuesToJson(param)
			request.Headers.Set("Content-Type", "application/json")
		}
		plainText += "api.huobi.pro\n"
		plainText += function + "\n"
		plainText += payload
		signature, err := HmacSign(SHA256, plainText, e.Option.SecretKey, true)
		payload += "&Signature=" + url.QueryEscape(signature)
		if err != nil {
			return
		}
		request.Url = fmt.Sprintf("%s%s", e.Option.RestPrivateHost, function) + "?" + payload
	}
	return request
}

func (e *HuobiRest) HandleError(request exchanges.Request, response []byte) error {
	type Result struct {
		Code      int
		Message   string
		Status    string
		ErrorCode string `json:"err-code"`
	}
	var result Result
	if err := json.Unmarshal(response, &result); err != nil {
		return err
	}
	if result.Code == 200 || result.Status == "ok" {
		return nil
	}
	errCode, ok := e.errors[result.ErrorCode]
	if ok {
		return wsex.ExError{Code: errCode, Message: result.Message}
	} else {
		return wsex.ExError{Code: wsex.UnHandleError, Message: fmt.Sprintf("code:%v msg:%v", result.Code, result.Message)}
	}
}
