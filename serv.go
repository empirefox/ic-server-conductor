package main

import (
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

var msgToSrcChan = make(chan []byte, 64)
var msgToDestChan = make(chan []byte, 64)
var hasSrc = false

func readPump(ws *websocket.Conn, to chan []byte, name string) {
	defer func() {
		ws.Close()
		glog.Infoln(name, "ws closed")
	}()
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		glog.Infoln("read from", name, string(message))
		to <- message
	}
}

func writePump(to *websocket.Conn, msgChan chan []byte, name string) {
	defer func() {
		to.Close()
		glog.Infoln(name, "ws closed")
	}()
	for msg := range msgChan {
		glog.Infoln("send to", name, string(msg))
		if err := to.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func serveSrc(ws *websocket.Conn) {
	defer func() {
		hasSrc = false
		ws.Close()
	}()
	hasSrc = true

	go readPump(ws, msgToDestChan, "src")

	// send msg to home
	writePump(ws, msgToSrcChan, "src")
}

func serveDest(ws *websocket.Conn) {
	defer ws.Close()

	go readPump(ws, msgToSrcChan, "dst")

	// send msg to home
	writePump(ws, msgToDestChan, "dst")

	// ping
	//	ticker := time.NewTicker(pingPeriod)
	//	defer ticker.Stop()
	//	for {
	//		select {
	//		case <-ticker.C:
	//			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
	//				return
	//			}
	//		}
	//	}
}
