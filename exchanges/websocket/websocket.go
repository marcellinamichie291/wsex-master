/*
@Time : 2021/4/13 10:46 上午
@Author : shiguantian
@File : websocket
@Software: GoLand
*/
package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Msg  []byte
	Type int
}

type WsConn struct {
	conn *websocket.Conn
	Options

	messageBufferChan chan Message
	stop              chan struct{}
	lock              sync.Mutex
	once              sync.Once
}

func (w *WsConn) Connect(options ...Option) (err error) {
	for _, o := range options {
		o(&w.Options)
	}

	if w.ReadDeadLineTime == 0 {
		w.ReadDeadLineTime = time.Minute
	}

	w.messageBufferChan = make(chan Message, 10)
	w.stop = make(chan struct{})

	if w.conn, err = w.connect(); err != nil {
		return err
	}

	return
}

func (w *WsConn) Close() {
	w.once.Do(func() {
		err := w.conn.Close()
		if err != nil {
			log.Printf("[WsConn] %s - close websocket error: %s", w.ExchangeName, err)
		}
		if w.closeHandler != nil {
			w.closeHandler(w.wsUrl)
		}
	})
}

func (w *WsConn) SendMessage(msg []byte) {
	w.messageBufferChan <- Message{Msg: msg, Type: websocket.TextMessage}
}

func (w *WsConn) SendJsonMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	w.messageBufferChan <- Message{Msg: data, Type: websocket.TextMessage}
	return nil
}

func (w *WsConn) SendPingMessage(msg []byte) {
	w.messageBufferChan <- Message{Msg: msg, Type: websocket.PingMessage}
}

func (w *WsConn) SendPongMessage(msg []byte) {
	w.messageBufferChan <- Message{Msg: msg, Type: websocket.PongMessage}
}

func (w *WsConn) SendCloseMessage(msg []byte) {
	w.messageBufferChan <- Message{Msg: msg, Type: websocket.CloseMessage}
}

func (w *WsConn) connect() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		EnableCompression: w.EnableCompression,
	}
	if w.ProxyUrl != "" {
		proxy, err := url.Parse(w.ProxyUrl)
		if err != nil {
			return nil, fmt.Errorf("[WsConn] %s - parse proxy url:%s error %s", w.ExchangeName, w.ProxyUrl, err)
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	conn, _, err := dialer.Dial(w.wsUrl, http.Header(w.ReqHeaders))
	if err != nil {
		return nil, fmt.Errorf("[WsConn] %s -  connect host: %s error:%s", w.ExchangeName, w.wsUrl, err)
	}

	w.stop = make(chan struct{})
	w.once = sync.Once{}
	go w.readLoop()
	go w.writeLoop()
	return conn, err
}

func (w *WsConn) reconnect() {
	w.lock.Lock()
	defer w.lock.Unlock()

	var err error

	var conn *websocket.Conn
	for retry := 0; retry < 20; retry++ {
		conn, err = w.connect()
		if err == nil {
			break
		}
		log.Printf("[WsConn] %s - reconnect failed: %s", w.ExchangeName, err)
		time.Sleep(time.Second * time.Duration(retry/4+1))
	}

	if err != nil {
		log.Printf("[WsConn] %s - try reconnect 20 times failed: %s", w.ExchangeName, err)
		w.Close()
		return
	}

	if w.reConnectHandler != nil {
		w.reConnectHandler(w.wsUrl)
	}
	w.conn = conn
}

func (w *WsConn) readLoop() {
	log.Printf("[WsConn] %s - start read loop\n", w.ExchangeName)

	w.conn.SetPingHandler(func(appData string) error {
		w.SendPongMessage([]byte(appData))
		w.conn.SetReadDeadline(time.Now().Add(w.ReadDeadLineTime))
		return nil
	})

	w.conn.SetPongHandler(func(appData string) error {
		w.conn.SetReadDeadline(time.Now().Add(w.ReadDeadLineTime))
		return nil
	})

	for {
		select {
		case <-w.stop:
			log.Printf("[WsConn] %s - websocket closed, exit read message loop", w.ExchangeName)
			return
		default:
			if w.conn == nil {
				log.Printf("[WsConn] %s - read message, no connection available:", w.ExchangeName)
				time.Sleep(time.Second)
				continue
			}
			w.conn.SetReadDeadline(time.Now().Add(w.ReadDeadLineTime))
			t, msg, err := w.conn.ReadMessage()
			if err != nil {
				log.Printf("[WsConn] %s - read message error:%s", w.ExchangeName, err)

				if w.disConnectedHandler != nil {
					w.disConnectedHandler(w.wsUrl, err)
				}

				close(w.stop)
				if w.IsAutoReconnect {
					w.reconnect()
				} else {
					w.Close()
				}
				return
			}
			if w.messageHandler == nil {
				return
			}
			switch t {
			case websocket.TextMessage:
				w.messageHandler(w.wsUrl, msg)
			case websocket.BinaryMessage:
				if w.decompressHandler == nil {
					w.messageHandler(w.wsUrl, msg)
				} else {
					msg2, err := w.decompressHandler(msg)
					if err != nil {
						if w.errorHandler != nil {
							w.errorHandler(w.wsUrl, fmt.Errorf("[WsConn] %s - decompress message error:%s", w.ExchangeName, err))
						}
					} else {
						w.messageHandler(w.wsUrl, msg2)
					}
				}
			}
		}
	}
}

func (w *WsConn) writeLoop() {
	log.Printf("[WsConn] %s - start write message\n", w.ExchangeName)
	if w.HeartbeatIntervalTime == 0 {
		w.HeartbeatIntervalTime = time.Hour
	}
	heartTimer := time.NewTimer(w.HeartbeatIntervalTime)
	for {
		select {
		case <-w.stop:
			log.Printf("[WsConn] %s - websocket closed, exit write message loop", w.ExchangeName)
			return
		case msg := <-w.messageBufferChan:
			err := w.conn.WriteMessage(msg.Type, msg.Msg)
			if err != nil {
				if w.errorHandler != nil {
					w.errorHandler(w.wsUrl, fmt.Errorf("[WsConn] %s - write message error:%s", w.ExchangeName, err))
				}
			}
		case <-heartTimer.C:
			if w.heartbeatHandler != nil {
				go w.heartbeatHandler(w.wsUrl)
				heartTimer.Reset(w.HeartbeatIntervalTime)
			}
		}
	}
}
