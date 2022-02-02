package huobi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shiguantian/wsex"
	. "github.com/shiguantian/wsex/utils"
)

type Response struct {
	Action string  `json:"action"`
	Topic  string  `json:"ch"`
	Rep    string  `json:"rep"`
	Ping   float64 `json:"ping"`
	Code   int     `json:"code"`
}

type OrderBookRes struct {
	Depth struct {
		SeqNum     float64       `json:"seqNum"`
		PrevSeqNum float64       `json:"prevSeqNum"`
		Asks       wsex.RawDepth `json:"asks"`
		Bids       wsex.RawDepth `json:"bids"`
	} `json:"tick" rep:"data"`
}

func (o OrderBookRes) parseOrderBook(symbol string) wsex.OrderBook {
	orderBook := wsex.OrderBook{}
	orderBook.Symbol = symbol
	for _, ask := range o.Depth.Asks {
		depthItem, err := ask.ParseRawDepthItem()
		if err != nil {
			continue
		}
		orderBook.Asks = append(orderBook.Asks, depthItem)
	}
	for _, bid := range o.Depth.Bids {
		depthItem, err := bid.ParseRawDepthItem()
		if err != nil {
			continue
		}
		orderBook.Bids = append(orderBook.Bids, depthItem)
	}
	sort.Sort(sort.Reverse(orderBook.Bids))
	sort.Sort(orderBook.Asks)
	return orderBook
}

// Order pushed by websocket api
type SymbolTicker struct {
	BuyPrice  float64 `json:"bid"`
	BuySize   float64 `json:"bidSize"`
	High      float64 `json:"high"`
	Last      float64 `json:"close"`
	Low       float64 `json:"low"`
	SellPrice float64 `json:"ask"`
	SellSize  float64 `json:"askSize"`
	Open      float64 `json:"open"`
	Vol       float64 `json:"vol"`
	SymbolId  string  `json:"symbol"`
}
type Ticker struct {
	Buy  []float64 `json:"bid"`
	High float64   `json:"high"`
	Last float64   `json:"close"`
	Low  float64   `json:"low"`
	Sell []float64 `json:"ask"`
	Open float64   `json:"open"`
	Vol  float64   `json:"vol"`
}
type TickerRes struct {
	Ticker    Ticker        `json:"tick"`
	Timestamp time.Duration `json:"ts"`
}

func (t TickerRes) parseTicker() wsex.Ticker {
	return wsex.Ticker{
		BestBuyPrice:   t.Ticker.Buy[0],
		BestSellPrice:  t.Ticker.Sell[1],
		Open:           t.Ticker.Open,
		Last:           t.Ticker.Last,
		High:           t.Ticker.High,
		Low:            t.Ticker.Low,
		Vol:            t.Ticker.Vol,
		Timestamp:      t.Timestamp,
		BestBuyAmount:  t.Ticker.Buy[1],
		BestSellAmount: t.Ticker.Sell[1],
	}
}

type AllTickerRes struct {
	Timestamp time.Duration  `json:"ts"`
	Data      []SymbolTicker `json:"data"`
}

func (ts AllTickerRes) parseAllTickers(symbolMap map[string]string) map[string]wsex.Ticker {
	tickers := make(map[string]wsex.Ticker)
	for _, t := range ts.Data {
		if symbol, ok := symbolMap[t.SymbolId]; ok {
			ticker := wsex.Ticker{
				BestBuyPrice:   t.BuyPrice,
				BestBuyAmount:  t.BuySize,
				BestSellPrice:  t.SellPrice,
				BestSellAmount: t.SellSize,
				Open:           t.Open,
				Last:           t.Last,
				Vol:            t.Vol,
				High:           t.High,
				Low:            t.Low,
				Timestamp:      ts.Timestamp,
			}
			tickers[symbol] = ticker
		}
	}
	return tickers
}

type Trade struct {
	Timestamp time.Duration `json:"ts"`
	Price     float64       `json:"price"`
	Amount    float64       `json:"amount"`
	Type      string        `json:"direction"`
}

type TradeRes struct {
	Trade struct {
		Data []Trade
	} `json:"tick"`
}

func (t Trade) parseTrade() wsex.Trade {
	var side wsex.Side = wsex.Buy
	if t.Type == "sell" {
		side = wsex.Sell
	}
	return wsex.Trade{
		Timestamp: t.Timestamp,
		Price:     t.Price,
		Amount:    t.Amount,
		Side:      side,
	}
}

type KLineRes struct {
	Data      []KlineItem   `json:"data"`
	Timestamp time.Duration `json:"ts"`
}

type KlineItem struct {
	Timestamp time.Duration `json:"id"`
	Open      float64       `json:"open"`
	Close     float64       `json:"close"`
	Low       float64       `json:"low"`
	High      float64       `json:"high"`
	Vol       float64       `json:"vol"`
}

func (t KLineRes) parseKLine(market wsex.Market, kLineType wsex.KLineType) []wsex.KLine {
	klines := []wsex.KLine{}
	for _, ele := range t.Data {
		kline := wsex.KLine{
			Symbol:    market.Symbol,
			Timestamp: ele.Timestamp,
			Type:      kLineType,
			Open:      ele.Open,
			Close:     ele.Close,
			High:      ele.High,
			Low:       ele.Low,
			Volume:    ele.Vol,
		}
		klines = append(klines, kline)
	}
	return klines
}

type Market struct {
	Symbol          string  `json:"symbol"`
	Base            string  `json:"base-currency"`
	Quote           string  `json:"quote-currency"`
	MinAmount       float64 `json:"min-order-amt"`
	AmountPrecision int     `json:"amount-precision"`
	PricePrecision  int     `json:"price-precision"`
}
type SymbolListRes struct {
	Data []Market `json:"data"`
}

type AccountRes struct {
	Data []struct {
		Id int `json:"id"`
	} `json:"data"`
}

type BalanceRes struct {
	Data struct {
		List []struct {
			Currency string `json:"currency"`
			Type     string `json:"type"`
			Balance  string `json:"balance"`
		} `json:"list"`
	} `json:"data"`
}

func (b BalanceRes) parseBalance() map[string]wsex.Balance {
	balances := make(map[string]wsex.Balance)
	for _, value := range b.Data.List {
		currency := strings.ToUpper(value.Currency)
		var balance wsex.Balance
		var ok bool
		if balance, ok = balances[currency]; !ok {
			balances[currency] = wsex.Balance{
				Asset: strings.ToUpper(currency),
			}
		}
		if value.Type == "trade" {
			balance.Available = SafeParseFloat(value.Balance)
		}
		if value.Type == "frozen" {
			balance.Frozen = SafeParseFloat(value.Balance)
		}
		balances[currency] = balance
	}
	return balances
}

// OrderInfo response by rest api
type Order struct {
	ID          int           `json:"id" open:"id" ws:"orderId"`
	Price       string        `json:"price" open:"price" ws:"orderPrice"`
	State       string        `json:"state" open:"state" ws:"orderStatus"`
	TotalAmount string        `json:"amount" open:"amount" ws:"orderSize"`
	TradeAmount string        `json:"field-amount" open:"filled-amount" ws:"tradeVolume"`
	TradeMoney  string        `json:"field-cash-amount" open:"filled-cash-amount"`
	TradeDate   time.Duration `ws:"tradeTime"`
	CreadeDate  time.Duration `json:"created-at" open:"created-at" ws:"orderCreateTime"`
	Type        string        `json:"type" open:"type" ws:"type"`
	ClientID    string        `json:"client-order-id" open:"client-order-id" ws:"clientOrderId"`
	FillPrice   string        `ws:"tradePrice"`
}
type OrderRes struct {
	Data Order `json:"data"`
}

func (o OrderRes) parseOrder(symbol string, market wsex.Market) wsex.Order {
	order := wsex.Order{
		ID:              strconv.Itoa(o.Data.ID),
		ClientID:        o.Data.ClientID,
		Symbol:          symbol,
		Price:           o.Data.Price,
		Amount:          o.Data.TotalAmount,
		Filled:          o.Data.TradeAmount,
		CreateTime:      o.Data.CreadeDate,
		TransactionTime: o.Data.TradeDate,
	}
	if o.Data.TradeMoney == "" {
		order.Cost = fmt.Sprintf("%v", SafeParseFloat(o.Data.TradeAmount)*SafeParseFloat(o.Data.FillPrice))
	} else {
		order.Cost = o.Data.TradeMoney
	}
	types := strings.Split(o.Data.Type, "-")
	switch types[0] {
	case "sell":
		order.Side = wsex.Sell
	case "buy":
		order.Side = wsex.Buy
	}
	switch strings.Join(types[1:], "-") {
	case "limit":
		order.Type = wsex.LIMIT
	case "market":
		order.Type = wsex.MARKET
	case "ioc":
		order.OrderType = wsex.IOC
	case "limit-fok":
	}
	switch o.Data.State {
	case "canceled":
		order.Status = wsex.Canceled
		filled, err := strconv.ParseFloat(order.Filled, 64)
		if err == nil && filled > 0 {
			order.Status = wsex.Close
		}
	case "filled":
		order.Status = wsex.Close
	case "created", "submitted":
		order.Status = wsex.Open
	case "partial-filled":
		order.Status = wsex.Partial
	default:
		order.Status = wsex.OrderStatusUnKnown
	}
	return order
}

type OpenOrderList struct {
	Data []Order `open:"data"`
}

type ResponseEvent struct {
	Channel string `json:"channel"`
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type OrderBook struct {
	wsex.OrderBook
	SeqNum     float64
	PrevSeqNum float64
}

func (o *OrderBook) update(bookData OrderBookRes) {
	o.Bids = o.Bids.Update(bookData.Depth.Bids, true)
	o.Asks = o.Asks.Update(bookData.Depth.Asks, false)
	o.SeqNum = bookData.Depth.SeqNum
	o.PrevSeqNum = bookData.Depth.PrevSeqNum
}

type SymbolOrderBook map[string]*OrderBook

type PingAction struct {
	Data struct {
		Timestamp time.Duration `json:"ts"`
	} `json:"data"`

	Action string `json:"action"`
}

type Balance struct {
	Data struct {
		Currency    string        `json:"currency"`
		Balance     string        `json:"balance"`
		AccountType string        `json:"accountType"`
		Timestamp   time.Duration `json:"changeTime"`
		Available   string        `json:"available"`
	}
}

type WsTickerRes struct {
	Ticker struct {
		Amount float64 `json:"amount"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Count  float64 `json:"count"`
		Vol    float64 `json:"vol"`
	} `json:"tick"`
	Timestamp time.Duration `json:"ts"`
	Topic     string        `json:"ch"`
}

func (t WsTickerRes) parseWsTicker(symbol string) wsex.Ticker {
	ticker := wsex.Ticker{
		Vol:       t.Ticker.Vol,
		Open:      t.Ticker.Open,
		Last:      t.Ticker.Close,
		High:      t.Ticker.High,
		Low:       t.Ticker.Low,
		Timestamp: t.Timestamp,
		Symbol:    symbol,
	}
	return ticker
}

type WsKlineRes struct {
	Ticker struct {
		Amount float64 `json:"amount"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Count  float64 `json:"count"`
		Vol    float64 `json:"vol"`
	} `json:"tick"`
	Timestamp time.Duration `json:"ts"`
	Topic     string        `json:"ch"`
}

func (k WsKlineRes) parseKline(symbol string) wsex.KLine {
	kline := wsex.KLine{
		Timestamp: k.Timestamp,
		Symbol:    symbol,
		Open:      k.Ticker.Open,
		Close:     k.Ticker.Close,
		Low:       k.Ticker.Low,
		High:      k.Ticker.High,
		Volume:    k.Ticker.Vol,
	}

	value := (strings.Split(k.Topic, "."))[3]
	switch value {
	case "1min":
		kline.Type = wsex.KLine1Minute
	case "5min":
		kline.Type = wsex.KLine5Minute
	case "15min":
		kline.Type = wsex.KLine15Minute
	case "30min":
		kline.Type = wsex.KLine30Minute
	case "60min":
		kline.Type = wsex.KLine1Hour
	case "4hour":
		kline.Type = wsex.KLine4Hour
	case "1day":
		kline.Type = wsex.KLine1Day
	case "1week":
		kline.Type = wsex.KLine1Week
	}
	return kline
}
