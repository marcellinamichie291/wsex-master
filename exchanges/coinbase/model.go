package coinbase

import (
	"sort"
	"strings"
	"time"

	"github.com/shiguantian/wsex"
	. "github.com/shiguantian/wsex/utils"
)

type Response struct {
	Type string `json:"type"`
}

type OrderBookRes struct {
	Asks   wsex.RawDepth `json:"asks"`
	Bids   wsex.RawDepth `json:"bids"`
	Symbol string        `json:"product_id"`
}

func (o OrderBookRes) parseOrderBook(symbol string) wsex.OrderBook {
	orderBook := wsex.OrderBook{}
	orderBook.Symbol = symbol
	for _, ask := range o.Asks {
		depthItem, err := ask.ParseRawDepthItem()
		if err != nil {
			continue
		}
		orderBook.Asks = append(orderBook.Asks, depthItem)
	}
	for _, bid := range o.Bids {
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

type Ticker struct {
	Buy  []float64 `json:"bid"`
	High float64   `json:"high"`
	Last float64   `json:"close"`
	Low  float64   `json:"low"`
	Sell []float64 `json:"ask"`
	Open float64   `json:"open"`
	Vol  float64   `json:"vol"`
}
type Stats24Hr struct {
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Last   string `json:"last"`
	Volume string `json:"volume"`
}
type AllTickerItem struct {
	Ticker Stats24Hr `json:"stats_24hour"`
}

type Trade struct {
	Timestamp string `json:"time"`
	Price     string `json:"price"`
	Amount    string `json:"size"`
	Type      string `json:"side"`
}

func (t Trade) parseTrade(symbol string) (trade wsex.Trade, err error) {
	ts := parseTime(t.Timestamp)
	var side wsex.Side = wsex.Buy
	if t.Type == "sell" {
		side = wsex.Sell
	}
	trade = wsex.Trade{
		Symbol:    symbol,
		Timestamp: ts,
		Price:     SafeParseFloat(t.Price),
		Amount:    SafeParseFloat(t.Amount),
		Side:      side,
	}
	return
}

type KlineRes [][]interface{}

func (t KlineRes) parseKLine(market wsex.Market, kLineType wsex.KLineType) []wsex.KLine {
	klines := []wsex.KLine{}
	for _, ele := range t {
		kline := wsex.KLine{
			Symbol:    market.Symbol,
			Timestamp: time.Duration(ele[0].(float64)),
			Type:      kLineType,
			Open:      ele[3].(float64),
			Close:     ele[4].(float64),
			High:      ele[2].(float64),
			Low:       ele[1].(float64),
			Volume:    ele[5].(float64),
		}
		klines = append(klines, kline)
	}
	return klines
}

type Market struct {
	Symbol          string `json:"id"`
	Base            string `json:"base_currency"`
	Quote           string `json:"quote_currency"`
	MinAmount       string `json:"base_min_size"`
	AmountPrecision string `json:"base_increment"`
	PricePrecision  string `json:"quote_increment"`
}
type SymbolListRes []Market

type OrderBook struct {
	wsex.OrderBook
	SeqNum     float64
	PrevSeqNum float64
}

type SymbolOrderBook map[string]*OrderBook

type WsTickerRes struct {
	Symbol        string `json:"product_id"`
	Price         string `json:"price"`
	Open          string `json:"open_24h"`
	High          string `json:"high_24h"`
	Low           string `json:"low_24h"`
	Vol           string `json:"volume_24h"`
	BestBidsPrice string `json:"best_bid"`
	BestAsksPrice string `json:"best_ask"`
	Time          string `json:"time"`
}

func (t WsTickerRes) parseTicker(market wsex.Market) wsex.Ticker {
	ts := parseTime(t.Time)
	ticker := wsex.Ticker{
		Vol:           SafeParseFloat(t.Vol),
		Open:          SafeParseFloat(t.Open),
		Last:          SafeParseFloat(t.Price),
		High:          SafeParseFloat(t.High),
		Low:           SafeParseFloat(t.Low),
		Timestamp:     ts,
		Symbol:        market.Symbol,
		BestBuyPrice:  SafeParseFloat(t.BestBidsPrice),
		BestSellPrice: SafeParseFloat(t.BestAsksPrice),
	}
	return ticker
}

type WsTradeRes struct {
	Side   string `json:"side"`
	Price  string `json:"price"`
	Symbol string `json:"product_id"`
	Size   string `json:"size"`
	Time   string `json:"time"`
}

func (t WsTradeRes) parseTrade(market wsex.Market) wsex.Trade {
	ts := parseTime(t.Time)
	var side wsex.Side = wsex.Buy
	if t.Side == "sell" {
		side = wsex.Sell
	}
	trade := wsex.Trade{
		Timestamp: ts,
		Symbol:    market.Symbol,
		Price:     SafeParseFloat(t.Price),
		Amount:    SafeParseFloat(t.Size),
		Side:      side,
	}
	return trade
}

type WsOrderBookUpdateRes struct {
	Symbol  string     `json:"product_id"`
	Changes [][]string `json:"changes"`
}

type CoinBaseOrderBook struct {
	wsex.OrderBook
}

func (o *CoinBaseOrderBook) update(bookData OrderBookRes) {
	o.Bids = o.Bids.Update(bookData.Bids, true)
	o.Asks = o.Asks.Update(bookData.Asks, false)
}

func parseTime(t string) (duration time.Duration) {
	timeTuple := strings.Split(t, ".")
	if len(timeTuple) == 2 {
		for 4-len(timeTuple[1]) > 0 {
			timeTuple[1] = timeTuple[1][:len(timeTuple[1])-1] + "0" + "Z"
		}
		if len(timeTuple[1])-4 > 0 {
			timeTuple[1] = timeTuple[1][:3] + "Z"
		}
	} else {
		timeTuple[0] = timeTuple[0][:len(timeTuple[0])-1]
		timeTuple = append(timeTuple, "000Z")
	}
	t = timeTuple[0] + "." + timeTuple[1]
	ts, err := time.Parse("2006-01-02T15:04:05.000Z", t)
	if err != nil {
		return
	}
	duration = time.Duration(ts.UnixNano() / 1000000)
	return
}
