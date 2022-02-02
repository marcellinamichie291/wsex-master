/*
@Time : 2021/2/24 10:29 上午
@Author : shiguantian
@File : model
@Software: GoLand
*/
package zb

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shiguantian/wsex"
	. "github.com/shiguantian/wsex/utils"
)

// ResponseEvent
type ResponseEvent struct {
	Channel string `json:"channel"`
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RawOrderBook struct {
	Symbol   string        `json:"market"`
	Bids     wsex.RawDepth `json:"listDown"`
	Asks     wsex.RawDepth `json:"listUp"`
	LastTime time.Duration `json:"lastTime"`
}

// OrderBook
type OrderBook struct {
	wsex.OrderBook
	LastTime time.Duration `json:"lastTime"` //
}

func (o *OrderBook) update(bookData RawOrderBook) {
	o.Bids = wsex.Depth{}
	o.Asks = wsex.Depth{}
	for _, b := range bookData.Bids {
		item, err := b.ParseRawDepthItem()
		if err != nil {
			continue
		}
		o.Bids = append(o.Bids, item)
	}
	sort.Sort(sort.Reverse(o.Bids))
	for _, a := range bookData.Asks {
		item, err := a.ParseRawDepthItem()
		if err != nil {
			continue
		}
		o.Asks = append(o.Asks, item)
	}
	sort.Sort(o.Asks)
	o.LastTime = bookData.LastTime
}

// OrderBook of one symbol
type SymbolOrderBook map[string]*OrderBook

// Order pushed by websocket api
type Order struct {
	Symbol string        `json:"market"`
	Record []interface{} `json:"record"`
}

// OrderInfo response by rest api
type OrderInfo struct {
	ID          string  `json:"id"`
	Price       float64 `json:"price"`
	Status      int     `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	TradeAmount float64 `json:"trade_amount"`
	TradeMoney  float64 `json:"trade_money"`
	TradeDate   int64   `json:"trade_date"`
	Type        int     `json:"type"`
}

type Ticker struct {
	Buy  string `json:"buy"`
	High string `json:"high"`
	Last string `json:"last"`
	Low  string `json:"low"`
	Sell string `json:"sell"`
	Open string `json:"open"`
	Vol  string `json:"vol"`
}

func (t Ticker) parseTicker() wsex.Ticker {
	return wsex.Ticker{
		BestBuyPrice:  SafeParseFloat(t.Buy),
		BestSellPrice: SafeParseFloat(t.Sell),
		Open:          SafeParseFloat(t.Open),
		Last:          SafeParseFloat(t.Last),
		High:          SafeParseFloat(t.High),
		Low:           SafeParseFloat(t.Low),
		Vol:           SafeParseFloat(t.Vol),
	}
}

type TickerRes struct {
	Ticker Ticker `json:"ticker"`
	Date   string `json:"date"`
}

func (t TickerRes) parseTicker() wsex.Ticker {
	ticker := t.Ticker.parseTicker()
	ticker.Timestamp = time.Duration(SafeParseFloat(t.Date))
	return ticker
}

type Trade struct {
	Date   float64 `json:"date"`
	Price  string  `json:"price"`
	Amount string  `json:"amount"`
	Type   string  `json:"type"`
}

func (t Trade) parseTrade() wsex.Trade {
	var side wsex.Side = wsex.Buy
	if t.Type == "sell" {
		side = wsex.Sell
	}
	return wsex.Trade{
		Timestamp: time.Duration(t.Date),
		Price:     SafeParseFloat(t.Price),
		Amount:    SafeParseFloat(t.Amount),
		Side:      side,
	}
}

type TradeRes struct {
	Trade []Trade `json:"data"`
}

type KLine struct {
	Data [][6]float64 `json:"data"`
}

type Balance struct {
	Currency  string `json:"showName"`
	Available string `json:"available"`
	Frozen    string `json:"freez"`
}

func (b Balance) parseBalance() wsex.Balance {
	return wsex.Balance{
		Asset:     strings.ToUpper(b.Currency),
		Available: SafeParseFloat(b.Available),
		Frozen:    SafeParseFloat(b.Frozen),
	}
}

type Balances struct {
	Timestamp float64   `json:"version"`
	Balances  []Balance `json:"coins"`
}

func parseSide(side int) wsex.Side {
	//0-limit sell,1-limit buy,2-PostOnly sell,3-PostOnly buy,4-IOC sell,5-IOC buy
	switch int(side) {
	case 0, 2, 4:
		return wsex.Sell
	case 1, 3, 5:
		return wsex.Buy
	default:
		return wsex.SideUnknown
	}
}

func parseStatus(status int, filled float64) wsex.OrderStatus {
	//1：canceled，2、filled/partial-canceled，0、3：opened/partial-filled
	switch status {
	case 1:
		if filled > 0 {
			return wsex.Close
		}
		return wsex.Canceled
	case 2:
		return wsex.Close
	case 0, 3:
		if filled > 0 {
			return wsex.Partial
		}
		return wsex.Open
	default:
		return wsex.OrderStatusUnKnown
	}
}

//Future

type FutureMarket struct {
	ID              string `json:"id"`
	MarketName      string `json:"marketName"`
	Symbol          string `json:"symbol"`
	PricePrecision  int    `json:"priceDecimal"`
	AmountPrecision int    `json:"amountDecimal"`
	Lot             string `json:"minAmount"`
	QuoteID         string `json:"buyerCurrencyName"`
	BaseID          string `json:"sellerCurrencyName"`
}

type FutureTicker []float64

func (t FutureTicker) parseTicker() wsex.Ticker {
	return wsex.Ticker{
		Open:      t[0],
		Last:      t[3],
		High:      t[1],
		Low:       t[2],
		Vol:       t[4],
		Timestamp: time.Duration(t[6]),
	}
}

type FutureTrade []float64

func (t FutureTrade) parseTrade() wsex.Trade {
	var side wsex.Side = wsex.Sell
	if t[2] == 1 {
		side = wsex.Buy
	}
	return wsex.Trade{
		Timestamp: time.Duration(t[3]),
		Price:     t[0],
		Amount:    t[1],
		Side:      side,
	}
}

type FutureKLine struct {
	Data [][]float64 `json:"data"`
}

type FutureOrder struct {
	ID            string `json:"id"`
	ClientOrderId string `json:"orderCode"`
	Price         string `json:"price"`
	Side          int    `json:"side"`
	Status        int    `json:"showStatus"`
	TotalAmount   string `json:"amount"`
	TradeAmount   string `json:"tradeAmount"`
	TradeMoney    string `json:"tradeValue"`
	Leverage      int    `json:"leverage"`
	TradeDate     string `json:"createTime"`
}

func (o FutureOrder) parseOrder(symbol string) (order wsex.Order) {
	order.ID = o.ID
	order.ClientID = o.ClientOrderId
	order.Price = o.Price
	order.Amount = o.TotalAmount
	order.Filled = o.TradeAmount
	order.Cost = o.TradeMoney
	order.Leverage = o.Leverage

	order.Symbol = symbol
	switch o.Side {
	case 1:
		order.Side = wsex.OpenLong
	case 2:
		order.Side = wsex.OpenShort
	case 3:
		order.Side = wsex.CloseLong
	case 4:
		order.Side = wsex.CloseShort
	}
	order.Status = wsex.OrderStatusUnKnown
	switch o.Status {
	case 1:
		order.Status = wsex.Open
	case 2:
		order.Status = wsex.Partial
	case 3, 7:
		order.Status = wsex.Close
	case 5:
		order.Status = wsex.Canceled
	}
	i, err := strconv.ParseInt(o.TradeDate, 10, 64)
	if err != nil {
		return
	}
	order.CreateTime = time.Duration(i)
	return
}

type FutureBalance struct {
	Currency  string `json:"currencyName" ws:"unit"`
	Available string `json:"amount" ws:"available"`
	Frozen    string `json:"freezeAmount" ws:"freeze"`
}

func (b FutureBalance) parseBalance() wsex.Balance {
	return wsex.Balance{
		Asset:     strings.ToUpper(b.Currency),
		Available: SafeParseFloat(b.Available),
		Frozen:    SafeParseFloat(b.Frozen),
	}
}

type FuturePosition struct {
	AvgPrice       string `json:"avgPrice"`       //开仓均价
	LiquidatePrice string `json:"liquidatePrice"` //强平价格
	Margin         string `json:"margin"`         //保证金
	MarginMode     int    `json:"marginMode"`     //逐仓，全仓
	MarginBalance  string `json:"marginBalance"`  //保证金余额
	MaintainMargin string `json:"maintainMargin"` //维持保证金
	Amount         string `json:"amount"`         //仓位数量
	FreezeAmount   string `json:"freezeAmount"`   //下单冻结仓位数量
	Side           int    `json:"side"`           //开多，开空
	Leverage       int    `json:"leverage"`       //杠杆倍数
	Symbol         string `json:"marketName"`     //marketName
	MarginRate     string `json:"marginRate"`
}

func (p FuturePosition) parsePositions(coin, symbol string) (positions wsex.FuturePositons) {

	switch p.Side {
	case 0:
		positions.PositionType = wsex.PositionShort
		positions.Amount = -SafeParseFloat(p.Amount)
	case 1:
		positions.PositionType = wsex.PositionLong
		positions.Amount = SafeParseFloat(p.Amount)
	}
	switch p.MarginMode {
	case 1:
		positions.MarginMode = wsex.FixedMargin
	case 2:
		positions.MarginMode = wsex.CrossedMargin
	}
	positions.Coin = coin
	positions.Symbol = symbol
	positions.AvgPrice = p.AvgPrice
	positions.FreezeAmount = p.FreezeAmount
	positions.Leverage = p.Leverage
	positions.LiquidatePrice = p.LiquidatePrice
	positions.Margin = p.Margin
	positions.MarginBalance = p.MarginBalance
	positions.MarginRate = p.MarginRate
	positions.MaintainMargin = p.MaintainMargin
	return positions
}

type FutureResponseEvent struct {
	Channel string `json:"channel"`
	Code    string `json:"errorCode"`
	Message string `json:"errorMsg"`
	Action  string `json:"action"`
}

type FutureAsset struct {
	Currency  string `json:"unit"`
	Available string `json:"available"`
	Frozen    string `json:"freeze"`
}

type FutureFundingRate struct {
	Rate          string `json:"fundingRate"`
	NextTimestamp string `json:"nextCalculateTime"`
}

func (f FutureFundingRate) parseFundingRate() (fundingRate wsex.FundingRate) {
	fundingRate.Rate = f.Rate
	t, err := time.Parse("2006-01-02 15:04:05", f.NextTimestamp)
	if err == nil {
		fundingRate.NextTimestamp = time.Duration(t.Unix())
	}
	return
}

//asset info
type FutureAssetInfo struct {
	AssetName string `json:"currencyName"`
	Amount    string `json:"amount"`
	Freeze    string `json:"freezeAmount"`
}

type FutureAccount struct {
	AccountBalance    string `json:"accountBalance"`    // Available + Freeze
	AccountNetBalance string `json:"accountNetBalance"` // AccountBalance + AllUnrealizedPnl
	Available         string `json:"available"`
	Freeze            string `json:"freeze"` // all frozen including position margin and open order margin
	PositionMargin    string `json:"allMargin"`
	AllUnrealizedPnl  string `json:"allUnrealizedPnl"`
}

//account info
type FutureAccountInfo struct {
	Account FutureAccount     `json:"account"`
	Assets  []FutureAssetInfo `json:"assets"`
}

func (f FutureAccountInfo) parseAccountInfo() (accountInfo wsex.FutureAccountInfo) {
	accountInfo.Account.Available = SafeParseFloat(f.Account.Available)
	accountInfo.Account.Total = SafeParseFloat(f.Account.AccountNetBalance)
	accountInfo.Account.Freeze = SafeParseFloat(f.Account.Freeze)
	accountInfo.Account.PositionMargin = SafeParseFloat(f.Account.PositionMargin)
	accountInfo.Account.OpenOrderMargin = accountInfo.Account.Freeze - accountInfo.Account.PositionMargin
	accountInfo.Account.AllUnrealizedPnl = SafeParseFloat(f.Account.AllUnrealizedPnl)
	accountInfo.Assets = make(map[string]wsex.FutureAsset)
	for _, ele := range f.Assets {
		asset := wsex.FutureAsset{
			AssetName: strings.ToUpper(ele.AssetName),
			Available: SafeParseFloat(ele.Amount),
			Freeze:    SafeParseFloat(ele.Freeze),
			// The single asset data returned by ZB does not have the following information.
			// Because it's only USDT margin currently, the information of account is shared
			Total:            SafeParseFloat(ele.Amount) + SafeParseFloat(ele.Freeze) + accountInfo.Account.AllUnrealizedPnl,
			AllUnrealizedPnl: accountInfo.Account.AllUnrealizedPnl,
			PositionMargin:   accountInfo.Account.PositionMargin,
			OpenOrderMargin:  accountInfo.Account.OpenOrderMargin,
		}
		accountInfo.Assets[asset.AssetName] = asset
	}
	return
}
