package main

import (
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

func main() {
    flag.Parse()
    log.SetFlags(0)

    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
    log.Printf("connecting to %s", u.String())

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
        }
    }()

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
	i := 0
	j := ""

    for {
        select {
        case <-done:
            return
        case t := <-ticker.C:
            fmt.Println(t.String())
			switch i {
			case 0:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_subscribe\",\"params\":[\"subscription\"]}")
			case 1:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getTrafficData\",\"params\":[{\"lines\":1}]}")
			case 2:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getDiskUsage\"}")
			case 3:
				j = string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getPeerList\"}")
			default:
				j = ""				
			}
			i++
            //j := string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getTrafficData\"}")
            //j := string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getDiskUsage\"}")
            //j := string("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"monitor_getPeerList\"}")
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




