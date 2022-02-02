/*
@Time : 2021/4/13 3:16 下午
@Author : shiguantian
@File : Option
@Software: GoLand
*/
package websocket

import "time"

// It will be invoked after websocket reconnected
type ReConnectedHandler func(url string)

// It will be invoked after websocket disconnected
type DisConnectedHandler func(url string, err error)

// It will be invoked after valid message received
type MessageHandler func(url string, message []byte)

// It will be invoked when websocket disconnected and reconnect failed.
type CloseHandler func(url string)

// It will be invoked when error happened
type ErrorHandler func(url string, err error)

// It will be invoked when data compression is set
type DecompressHandler func([]byte) ([]byte, error)

// It will be invoked When heartbeat is needed
type HeartbeatHandler func(url string)

type Options struct {
	ExchangeName          string
	wsUrl                 string
	ProxyUrl              string
	ReqHeaders            map[string][]string
	HeartbeatIntervalTime time.Duration
	ReadDeadLineTime      time.Duration

	IsAutoReconnect   bool
	EnableCompression bool

	reConnectHandler    ReConnectedHandler
	disConnectedHandler DisConnectedHandler
	messageHandler      MessageHandler
	errorHandler        ErrorHandler
	closeHandler        CloseHandler
	decompressHandler   DecompressHandler
	heartbeatHandler    HeartbeatHandler
}

type Option func(*Options)

func SetWsUrl(url string) Option {
	return func(o *Options) {
		o.wsUrl = url
	}
}

func SetExchangeName(name string) Option {
	return func(o *Options) {
		o.ExchangeName = name
	}
}

func SetProxyUrl(url string) Option {
	return func(o *Options) {
		o.ProxyUrl = url
	}
}

func SetReqHeaders(key, value string) Option {
	return func(o *Options) {
		o.ReqHeaders[key] = append(o.ReqHeaders[key], value)
	}
}

func SetHeartbeatIntervalTime(t time.Duration) Option {
	return func(o *Options) {
		o.HeartbeatIntervalTime = t
	}
}

func SetReadDeadLineTime(t time.Duration) Option {
	return func(o *Options) {
		o.ReadDeadLineTime = t
	}
}

func SetIsAutoReconnect(isAuto bool) Option {
	return func(o *Options) {
		o.IsAutoReconnect = isAuto
	}
}

func SetEnableCompression(enable bool) Option {
	return func(o *Options) {
		o.EnableCompression = enable
	}
}

func SetReConnectedHandler(handler ReConnectedHandler) Option {
	return func(o *Options) {
		o.reConnectHandler = handler
	}
}

func SetDisConnectedHandler(handler DisConnectedHandler) Option {
	return func(o *Options) {
		o.disConnectedHandler = handler
	}
}

func SetMessageHandler(handler MessageHandler) Option {
	return func(o *Options) {
		o.messageHandler = handler
	}
}

func SetErrorHandler(handler ErrorHandler) Option {
	return func(o *Options) {
		o.errorHandler = handler
	}
}

func SetCloseHandler(handler CloseHandler) Option {
	return func(o *Options) {
		o.closeHandler = handler
	}
}

func SetDecompressHandler(handler DecompressHandler) Option {
	return func(o *Options) {
		o.decompressHandler = handler
	}
}

func SetHeartbeatHandler(handler HeartbeatHandler) Option {
	return func(o *Options) {
		o.heartbeatHandler = handler
	}
}
