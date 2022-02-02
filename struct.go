/*
@Time : 2021/4/24 10:23 上午
@Author : shiguantian
@File : struct
@Software: GoLand
*/
package wsex

import (
	"errors"
	"fmt"
	"sort"
	"time"

	. "github.com/shiguantian/wsex/utils"
)

// Define unified data structure
type ExchangeType string

const (
	Binance ExchangeType = "binance"
	ZB                   = "zb"
	Okex                 = "okex"
	Huobi                = "huobipro"
	GateIo               = "gateio"
)

// Options
type Options struct {
	ExchangeName string // exchange name
	SecretKey    string // SecretKey key of this exchange account
	AccessKey    string // AccessKey key of this exchange account
	PassPhrase   string // Some exchanges need passphrase, like okex

	WsHost          string // websocket api host,  the default value will be used if not set
	RestHost        string // rest public api host,  the default value will be used if not set
	RestPrivateHost string // rest private api host,  the default value will be used if not set

	// the all markets of this exchange， key is the Market.Symbol.
	// As a config item, use to prevent high frequency requests causing restricted access when multiple instances are deployed on one server
	// if not set, the rest API will be called to get the market data
	Markets map[string]Market

	AutoReconnect       bool   // whether enable auto reconnect
	ProxyUrl            string // proxy, http://host:port
	ClientOrderIDPrefix string // Prefix of client order id，len better(0~10)
}

type FutureOptions struct {
	ContractType      ContractType
	FutureAccountType FutureAccountType
	FuturesKind       FuturesKind
}

type Market struct {
	SymbolID        string  // the market id of exchange, Each exchange has its own definition
	Symbol          string  // the unified market id: XXX/YYY
	BaseID          string  // sell coin, eg: MarketID = btcusdt, baseID = btc
	QuoteID         string  // buy coin, eg: MarketID = btcusdt, quoteID = usdt
	PricePrecision  int     // price precision
	AmountPrecision int     // amount precision
	Lot             float64 // min size
}

func (m Market) String() string {
	return fmt.Sprintf("%s/%s", m.BaseID, m.QuoteID)
}

type RawDepthItem []interface{}
type RawDepth []RawDepthItem

func (r RawDepthItem) ParseRawDepthItem() (item DepthItem, err error) {
	if len(r) < 2 {
		return item, errors.New("invalid data")
	}
	switch r[0].(type) {
	case float64:
		item.Price = fmt.Sprintf("%v", r[0].(float64))
		item.Amount = fmt.Sprintf("%v", r[1].(float64))
	case string:
		item.Price = r[0].(string)
		item.Amount = r[1].(string)
	default:
		err = errors.New("invalid data, type not support")
	}
	return
}

// DepthItem : each level data of the order book
type DepthItem struct {
	Price  string `json:"price"`
	Amount string `json:"amount"`
}

type Depth []DepthItem

func (d Depth) Len() int           { return len(d) }
func (d Depth) Less(i, j int) bool { return CompareFloatString(d[i].Price, d[j].Price) == CompareLess }
func (d Depth) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d Depth) Sort()              { sort.Sort(d) }
func (d Depth) Search(price string, reverse bool) int {
	index := -1
	i, j := 0, len(d)
	for i < j {
		h := int(uint(i+j) >> 1)
		ret := CompareFloatString(d[h].Price, price)
		if ret == CompareLess {
			if reverse {
				j = h
			} else {
				i = h + 1
			}
		} else if ret == CompareGreater {
			if reverse {
				i = h + 1
			} else {
				j = h
			}
		} else if ret == CompareEqual {
			index = h
			break
		} else {
			break
		}
	}
	return index
}

func (d Depth) RemoveByIndex(index int) Depth {
	if index >= 0 {
		return append(d[:index], d[index+1:]...)
	}
	return d
}

func (d Depth) Update(newDepth RawDepth, reverse bool) Depth {
	for _, rawItem := range newDepth {
		item, err := rawItem.ParseRawDepthItem()
		if err != nil {
			continue
		}
		index := d.Search(item.Price, reverse)
		amount := SafeParseFloat(item.Amount)
		if index >= 0 {
			if amount < ZERO {
				d = d.RemoveByIndex(index)
			} else {
				d[index] = item
			}
		} else {
			if amount > ZERO {
				d = append(d, item)
			}
		}
	}
	if reverse {
		sort.Sort(sort.Reverse(d))
	} else {
		sort.Sort(d)
	}
	l := len(d)
	if l > 400 {
		l = 400
	}
	return d[:l]
}

type OrderBook struct {
	Symbol string
	Bids   Depth `json:"bids"`
	Asks   Depth `json:"asks"`
}

func (o *OrderBook) Sort() {
	sort.Sort(sort.Reverse(o.Bids))
	sort.Sort(o.Asks)
}

type (
	KLineType   int
	Side        string
	TradeType   string
	OrderType   int
	OrderStatus string
)

const (
	KLineUnknown KLineType = iota
	KLine1Minute
	KLine3Minute
	KLine5Minute
	KLine15Minute
	KLine30Minute
	KLine1Hour
	KLine2Hour
	KLine4Hour
	KLine6Hour
	KLine8Hour
	KLine12Hour
	KLine1Day
	KLine3Day
	KLine1Week
	KLine1Month
)

const (
	SideUnknown Side = "Unknown"
	Buy              = "BUY"
	Sell             = "SELL"
	OpenLong         = "OpenLong"
	OpenShort        = "OpenShort"
	CloseLong        = "CloseLong"
	CloseShort       = "CloseShort"
)
const (
	TradeTypeUnKnown TradeType = "Unknown"
	LIMIT                      = "Limit"
	MARKET                     = "market"
)

const (
	OrderTypeUnKnown OrderType = -1
	Normal                     = iota // Normal order
	PostOnly                          // Post only, maker order
	FOK                               // Fill or Kill
	IOC                               // Immediate Or Cancel
)

const (
	OrderStatusUnKnown OrderStatus = "Unknown"
	Open                           = "open"
	Partial                        = "partial"
	Close                          = "close"
	Canceled                       = "canceled"
)

type Ticker struct {
	Symbol         string
	Timestamp      time.Duration
	BestBuyPrice   float64
	BestSellPrice  float64
	BestBuyAmount  float64
	BestSellAmount float64
	Open           float64
	Last           float64
	High           float64
	Low            float64
	Vol            float64
}

type Trade struct {
	Symbol    string
	Timestamp time.Duration
	Price     float64
	Amount    float64
	Side      Side
}

type KLine struct {
	Symbol    string
	Timestamp time.Duration
	Type      KLineType
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64
}

type Order struct {
	ID              string
	ClientID        string
	Symbol          string
	Price           string
	Amount          string
	Filled          string
	Cost            string
	Leverage        int
	Status          OrderStatus
	Side            Side
	Type            TradeType
	OrderType       OrderType
	CreateTime      time.Duration
	TransactionTime time.Duration
}

type Balance struct {
	Asset     string
	Available float64
	Frozen    float64
}

type BalanceUpdate struct {
	UpdateTime time.Duration
	Balances   map[string]Balance
}

type ContractType string

const (
	Swap    ContractType = "Swap"    //永续
	Futures              = "Futures" //交割
)

type FuturesKind string //交割种类

const (
	CurrentWeek    = "CurrentWeek"    //当周
	NextWeek       = "NextWeek"       //次周
	CurrentMonth   = "CurrentMonth"   //当月
	NextMonth      = "NextMonth"      //次月
	CurrentQuarter = "CurrentQuarter" //当季
	NextQuarter    = "NextQuarter"    //次季
)

type FutureMarginMode string

const (
	FixedMargin   FutureMarginMode = "FixedMargin"   //逐仓
	CrossedMargin                  = "CrossedMargin" //全仓
)

type FuturePositionsMode string

const (
	OneWay FuturePositionsMode = "OneWay" //单向持仓
	TwoWay                     = "TwoWay" //双向持仓
)

type FutureAccountType string

const (
	CoinMargin FutureAccountType = "CoinMargin" //币本位
	UsdtMargin                   = "UsdtMargin" //U本位
)

//asset info
type FutureAsset struct {
	AssetName        string
	Total            float64 // = Available + Freeze + AllUnrealizedPnl
	Available        float64 // available balance amount
	Freeze           float64 // = PositionMargin + OpenOrderMargin
	PositionMargin   float64 // frozen by position margin
	OpenOrderMargin  float64 // frozen by open order margin
	AllUnrealizedPnl float64 // unrealized profit
}

//account info
type FutureAccountInfo struct {
	Account   FutureAsset                                // account summary assets
	Assets    map[string]FutureAsset                     // key:AssetName
	Positions map[string]map[PositionType]FuturePositons // key:AssetName
}

//仓位信息
type FuturePositons struct {
	Coin           string           // 币名
	Symbol         string           //币对
	AvgPrice       string           //开仓均价
	LiquidatePrice string           //强平价格
	Margin         string           //保证金
	MarginMode     FutureMarginMode //逐仓，全仓
	MarginBalance  string           //保证金余额
	Amount         float64          //仓位数量
	FreezeAmount   string           //下单冻结仓位数量
	PositionType   PositionType     //开多，开空
	Leverage       int              //杠杆倍数
	MarginRate     string           //保证金率
	MaintainMargin string           //维持保证金
}

type FuturePositonsUpdate struct {
	Symbol   string
	Positons []FuturePositons
}

type MarkPrice struct {
	Symbol string
	Price  string
}

type PositionType string

const (
	PositionTypeUnKonwn PositionType = "PositionTypeUnKonw"
	PositionLong                     = "PositionLong"
	PositionShort                    = "PositionShort"
)

type FundingRate struct {
	Rate          string
	NextTimestamp time.Duration
}
