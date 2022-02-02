package coinbase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shiguantian/wsex"
	"github.com/shiguantian/wsex/exchanges"
	. "github.com/shiguantian/wsex/utils"
)

type CoinBaseRest struct {
	exchanges.BaseExchange
}

var AccountId int = 0

func (e *CoinBaseRest) Init(option wsex.Options) {
	e.Option = option

	if e.Option.RestHost == "" {
		e.Option.RestHost = "https://api.pro.coinbase.com"
	}
	if e.Option.RestPrivateHost == "" {
		e.Option.RestPrivateHost = "https://api.pro.coinbase.com"
	}
}

func (e *CoinBaseRest) FetchOrderBook(symbol string, size int) (orderBook wsex.OrderBook, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	params.Set("level", strconv.Itoa(size))
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/products/"+market.SymbolID+"/book", params, http.Header{})
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

func (e *CoinBaseRest) FetchTicker(symbol string) (ticker wsex.Ticker, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/products/"+market.SymbolID+"/stats", params, http.Header{})
	if err != nil {
		return
	}

	var data Stats24Hr
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	params.Set("level", "1")
	res, err = e.Fetch(e, exchanges.Public, exchanges.GET, "/products/"+market.SymbolID+"/book", params, http.Header{})
	if err != nil {
		return
	}
	var orderBookRes OrderBookRes
	if err = json.Unmarshal(res, &orderBookRes); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	bestBidsItem, err := orderBookRes.Bids[0].ParseRawDepthItem()
	if err != nil {
		return
	}
	bestAsksItem, err := orderBookRes.Asks[0].ParseRawDepthItem()
	if err != nil {
		return
	}
	ticker = wsex.Ticker{
		Symbol:         symbol,
		Timestamp:      time.Duration(time.Now().Unix()),
		BestBuyPrice:   SafeParseFloat(bestBidsItem.Price),
		BestSellPrice:  SafeParseFloat(bestAsksItem.Price),
		BestBuyAmount:  SafeParseFloat(bestBidsItem.Amount),
		BestSellAmount: SafeParseFloat(bestAsksItem.Amount),
		Open:           SafeParseFloat(data.Open),
		Last:           SafeParseFloat(data.Last),
		High:           SafeParseFloat(data.High),
		Low:            SafeParseFloat(data.Low),
		Vol:            SafeParseFloat(data.Volume),
	}
	return
}

func (e *CoinBaseRest) FetchAllTicker() (tickers map[string]wsex.Ticker, err error) {
	params := url.Values{}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/products/stats", params, http.Header{})
	if err != nil {
		return
	}
	var data map[string]AllTickerItem
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	tickers = make(map[string]wsex.Ticker)
	for symbol, t := range data {
		market, err := e.GetMarketByID(symbol)
		if err != nil {
			continue
		}
		tickers[market.Symbol] = wsex.Ticker{
			Open:   SafeParseFloat(t.Ticker.Open),
			Last:   SafeParseFloat(t.Ticker.Last),
			High:   SafeParseFloat(t.Ticker.High),
			Low:    SafeParseFloat(t.Ticker.Low),
			Vol:    SafeParseFloat(t.Ticker.Volume),
			Symbol: market.Symbol,
		}
	}
	return
}

func (e *CoinBaseRest) FetchTrade(symbol string) (trades []wsex.Trade, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/products/"+market.SymbolID+"/trades", params, http.Header{})
	if err != nil {
		return
	}
	//println(string(res))
	var data []Trade
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	for _, t := range data {
		trade, err := t.parseTrade(symbol)
		if err == nil {
			trades = append(trades, trade)
		}
	}
	return
}

func (e *CoinBaseRest) FetchKLine(symbol string, t wsex.KLineType) (klines []wsex.KLine, err error) {
	market, err := e.GetMarket(symbol)
	if err != nil {
		return
	}
	params := url.Values{}
	switch t {
	case wsex.KLine15Minute:
		params.Set("granularity", "900")
	case wsex.KLine1Day:
		params.Set("granularity", "86400")
	case wsex.KLine1Minute:
		params.Set("granularity", "60")
	case wsex.KLine1Hour:
		params.Set("granularity", "3600")
	case wsex.KLine6Hour:
		params.Set("granularity", "21600")
	case wsex.KLine5Minute:
		params.Set("granularity", "300")
	default:
		return nil, errors.New("coinbase can not support kline interval")
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/products/"+market.SymbolID+"/candles", params, http.Header{})
	if err != nil {
		return
	}
	var data KlineRes
	if err = json.Unmarshal(res, &data); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return
	}
	klines = data.parseKLine(market, t)
	return
}

func (e *CoinBaseRest) FetchMarkets() (map[string]wsex.Market, error) {
	if len(e.Option.Markets) > 0 {
		return e.Option.Markets, nil
	}
	res, err := e.Fetch(e, exchanges.Public, exchanges.GET, "/products", url.Values{}, http.Header{})
	if err != nil {
		return e.Option.Markets, err
	}

	var markets SymbolListRes
	if err = json.Unmarshal(res, &markets); err != nil {
		err = wsex.ExError{Code: wsex.ErrDataParse, Message: err.Error()}
		return e.Option.Markets, err
	}

	e.Option.Markets = make(map[string]wsex.Market)
	for _, value := range markets {
		market := wsex.Market{
			SymbolID: value.Symbol,
			Symbol:   strings.ToUpper(fmt.Sprintf("%v/%v", value.Base, value.Quote)),
			BaseID:   strings.ToUpper(value.Base),
			QuoteID:  strings.ToUpper(value.Quote),
			Lot:      SafeParseFloat(value.MinAmount),
		}
		pres := strings.Split(value.PricePrecision, ".")
		if len(pres) == 1 {
			market.PricePrecision = 0
		} else {
			market.PricePrecision = len(pres[1])
		}

		pres = strings.Split(value.AmountPrecision, ".")
		if len(pres) == 1 {
			market.AmountPrecision = 0
		} else {
			market.AmountPrecision = len(pres[1])
		}
		e.Option.Markets[market.Symbol] = market
	}
	return e.Option.Markets, nil
}
func (e *CoinBaseRest) FetchBalance() (balances map[string]wsex.Balance, err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *CoinBaseRest) CreateOrder(symbol string, price, amount float64, side wsex.Side, tradeType wsex.TradeType, orderType wsex.OrderType, useClientID bool) (order wsex.Order, err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *CoinBaseRest) CancelOrder(symbol, orderID string) (err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *CoinBaseRest) CancelAllOrders(symbol string) (err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *CoinBaseRest) FetchOrder(symbol, orderID string) (order wsex.Order, err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *CoinBaseRest) FetchOpenOrders(symbol string, pageIndex, pageSize int) (orders []wsex.Order, err error) {
	err = wsex.ExError{Code: wsex.NotImplement}
	return
}

func (e *CoinBaseRest) Sign(access, method, function string, param url.Values, header http.Header) (request exchanges.Request) {
	request.Headers = header
	request.Method = method
	if access == exchanges.Public {
		request.Url = fmt.Sprintf("%s%s", e.Option.RestHost, function)
		if len(param) > 0 {
			request.Url = request.Url + "?" + param.Encode()
		}
	} else {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		key, err := base64.StdEncoding.DecodeString(e.Option.SecretKey)
		if err != nil {
			return
		}
		signature := hmac.New(sha256.New, key)
		_, err = signature.Write([]byte(fmt.Sprintf(
			"%s%s%s%s",
			timestamp,
			method,
			function,
			UrlValuesToJson(param),
		)))
		if err != nil {
			return
		}
		request.Headers.Set("Content-Type", "application/json")
		request.Headers.Set("CB-ACCESS-KEY", e.Option.AccessKey)
		request.Headers.Set("CB-ACCESS-PASSPHRASE", e.Option.PassPhrase)
		request.Headers.Set("CB-ACCESS-TIMESTAMP", timestamp)
		request.Headers.Set("CB-ACCESS-SIGN", base64.StdEncoding.EncodeToString(signature.Sum(nil)))
		request.Url = fmt.Sprintf("%s%s", e.Option.RestPrivateHost, function)
		if len(param) > 0 {
			request.Url = request.Url + "?" + param.Encode()
		}
	}
	return request
}

func (e *CoinBaseRest) HandleError(request exchanges.Request, response []byte) error {
	type Result struct {
		Message string
	}
	var result Result
	if err := json.Unmarshal(response, &result); err != nil {
		return nil
	}
	if result.Message == "" {
		return nil
	}
	return wsex.ExError{Message: result.Message}
}
