package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:5678", "http service address")
var test_step int64
var subid string

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		return
	}
	flag.Parse()
	log.SetFlags(0)
	test_step = 0
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	cert, err := os.ReadFile("cert.pem")
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(cert); !ok {
		log.Fatalf("unable to parse cert from %s", "cert.pem")
	}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{
		MinVersion:               tls.VersionTLS13,
		PreferServerCipherSuites: true,
		RootCAs:                  certPool,
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("<-- %s", message)
			if test_step == 1 {
				var rsp jsonrpcMessage
				// Handle rsp
				if err = json.Unmarshal(message, &rsp); err == nil {
					var res string
					_ = json.Unmarshal(rsp.Result, &res)
					subid = res
					test_step++
				}
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	j := ""

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			switch test_step {
			case 0:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_subscribe\",\"params\":[\"subscription\", \"" + os.Args[1] + "\"]}")
			case 2:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getTrafficData\",\"params\":[{\"subid\":\"" + subid + "\",\"lines\":1}]}")
			case 3:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getDiskUsage\",\"params\":[{\"subid\":\"" + subid + "\"}]}")
			case 4:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getPeerList\",\"params\":[{\"subid\":\"" + subid + "\"}]}")
			default:
				j = ""
			}
			test_step++
			if j != "" {
				fmt.Println("-->", j)
				err := c.WriteMessage(websocket.TextMessage, []byte(j))
				if err != nil {
					log.Println("write:", err)
					return
				}
			}

		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
