/*
@Time : 2021/4/16 11:39 上午
@Author : shiguantian
@File : error
@Software: GoLand
*/
package wsex

import "fmt"

type ExError struct {
	Code    int                    // error code
	Message string                 // error message
	Data    map[string]interface{} // business data
}

func (e ExError) Error() string {
	return fmt.Sprintf("code: %v message: %v", e.Code, e.Message)
}

const (
	NotImplement = 10000 + iota
	UnHandleError

	//exchange api business error
	ErrExchangeSystem = 20000 + iota
	ErrDataParse
	ErrAuthFailed
	ErrRequestParams
	ErrInsufficientFunds
	ErrInvalidOrder
	ErrInvalidAddress
	ErrOrderNotFound
	ErrNotFoundMarket
	ErrChannelNotExist
	ErrInvalidDepth // recv dirty order book data

	//network error
	ErrDDoSProtection = 30000 + iota
	ErrTimeout
	ErrBadRequest
	ErrBadResponse
)
