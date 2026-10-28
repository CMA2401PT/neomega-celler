// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":2401", "http service address")

type actionAndData struct {
	action string
	data   []byte
}

func (d *actionAndData) String() string {
	cut := d.data
	if len(d.data) > 16 {
		cut = d.data[:16]
	}
	return fmt.Sprintf("%v:%d bytes %X", d.action, len(d.data), cut)
}

type WebsocketViewerServer struct {
	recvChan chan *actionAndData
	sendChan chan *actionAndData
}

func (vs *WebsocketViewerServer) _parseHead(data []byte) (string, []byte) {
	headLen := data[0]
	head := data[1 : 1+headLen]
	return string(head), data[1+headLen:]
}

func (vs *WebsocketViewerServer) Handle(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		CheckOrigin:       func(r *http.Request) bool { return true },
		EnableCompression: true,
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	recvChan := make(chan *actionAndData, 16)
	sendChan := make(chan *actionAndData, 16)
	defer c.Close()
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			action, data := vs._parseHead(message)
			recvChan <- &actionAndData{action: action, data: data}
		}
	}()
	go func() {
		for dataToSend := range sendChan {
			action := []byte(dataToSend.action)
			data := dataToSend.data
			combined := append(append(([]byte{byte(len(action))}), action...), data...)
			c.WriteMessage(websocket.BinaryMessage, combined)
		}
	}()
	vs.serveClient(recvChan, sendChan)
}

func (vs *WebsocketViewerServer) serveClient(recvChan chan *actionAndData, sendChan chan *actionAndData) {
	server := WorldFn[uint32](DefaultWorldFn[uint32]).ToStaticLambdaServer(time.Second / 10)
	go func() {
		sendTicker := time.NewTicker(time.Second / 10)
		datas := []SubchunkResponse[uint32]{}
		for {
			select {
			case changes := <-server.Changes:
				datas = append(datas, changes...)
			case <-sendTicker.C:
				if len(datas) == 0 {
					continue
				}
				dataToSend := ValueDumpFnUint32(uint32(len(datas)), nil)
				for _, c := range datas {
					dataToSend = c.Dump(dataToSend)
				}
				datas = []SubchunkResponse[uint32]{}

				sendChan <- &actionAndData{
					action: "sections",
					data:   dataToSend,
				}
			}
		}
	}()

	for actionAndData := range recvChan {
		switch actionAndData.action {
		case "updateCenter":
			p := unsafe.Pointer(&actionAndData.data[0])
			currentCenter := *((*CenteredArea)(p))
			fmt.Println(currentCenter)
			server.UpdateCenter(currentCenter)
		}
	}
}

func main() {
	flag.Parse()
	vs := &WebsocketViewerServer{}
	http.HandleFunc("/server", vs.Handle)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
