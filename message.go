/*
@Time : 2021/4/9 4:59 下午
@Author : shiguantian
@File : message
@Software: GoLand
*/
package wsex

type MessageType int

const (
	MsgOrderBook MessageType = iota
	MsgTicker
	MsgAllTicker
	MsgTrade
	MsgKLine
	MsgBalance
	MsgOrder
	MsgPositions
	MsgMarkPrice

	MsgReConnected
	MsgDisConnected
	MsgClosed
	MsgError
)

type Message struct {
	Type MessageType
	Data interface{}
}
type MessageChan chan Message

var (
	ReConnectedMessage  = Message{Type: MsgReConnected}
	DisConnectedMessage = Message{Type: MsgDisConnected}
	CloseMessage        = Message{Type: MsgClosed}
	ErrorMessage        = func(err error) Message { return Message{Type: MsgError, Data: err} }
)
