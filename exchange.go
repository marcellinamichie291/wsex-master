/*
@Time : 2021/4/16 4:24 下午
@Author : shiguantian
@File : exchange
@Software: GoLand
*/
package wsex

type IExchange interface {
	//websocket api
	SubscribeOrderBook(symbol string, level, speed int, isIncremental bool, sub MessageChan) (string, error)

	SubscribeTrades(symbol string, sub MessageChan) (string, error)

	SubscribeTicker(symbol string, sub MessageChan) (string, error)

	SubscribeAllTicker(sub MessageChan) (string, error)

	SubscribeKLine(symbol string, t KLineType, sub MessageChan) (string, error)

	SubscribeBalance(symbol string, sub MessageChan) (string, error)

	SubscribeOrder(symbol string, sub MessageChan) (string, error)

	UnSubscribe(topics string, sub MessageChan) error

	//rest api
	FetchOrderBook(symbol string, size int) (OrderBook, error)

	FetchTicker(symbol string) (Ticker, error)

	FetchAllTicker() (map[string]Ticker, error)

	FetchTrade(symbol string) ([]Trade, error)

	FetchKLine(symbol string, t KLineType) ([]KLine, error)

	FetchMarkets() (map[string]Market, error)

	FetchBalance() (map[string]Balance, error)

	CreateOrder(symbol string, price, amount float64, side Side, tradeType TradeType, orderType OrderType, useClientID bool) (Order, error)

	CancelOrder(symbol, orderID string) error

	CancelAllOrders(symbol string) error

	FetchOrder(symbol, orderID string) (Order, error)

	FetchOpenOrders(symbol string, pageIndex, pageSize int) ([]Order, error)
}

type IFutureExchange interface {
	IExchange

	Setting(symbol string, leverage int, marginMode FutureMarginMode, positionMode FuturePositionsMode) error

	FetchMarkPrice(symbol string) (MarkPrice, error)

	FetchFundingRate(symbol string) (FundingRate, error)

	FetchAccountInfo() (FutureAccountInfo, error)

	FetchPositions(symbol string) (positions []FuturePositons, err error)

	FetchAllPositions() (positions []FuturePositons, err error)

	SubscribePositions(symbol string, sub MessageChan) (string, error)

	SubscribeMarkPrice(symbol string, sub MessageChan) (string, error)
}
