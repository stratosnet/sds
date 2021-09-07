package httpserv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stratosnet/sds/utils"
)

// HTTPServ Http server
type HTTPServ struct {
	routeMap map[string]*httpHandler
	timeout  time.Duration
	headers  map[string]string
	port     int
}

// NewHTTPServ
func NewHTTPServ() *HTTPServ {
	return &HTTPServ{
		routeMap: make(map[string]*httpHandler),
		timeout:  10 * time.Second,
		headers:  make(map[string]string),
		port:     9608,
	}
}

type httpHandler struct {
	fh      funcHandler
	headers map[string]string
}

type funcHandler func(request *http.Request) []byte

func (hh *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	for k, v := range hh.headers {
		w.Header().Set(k, v)
	}
	w.Write(hh.fh(r))
}

// Route
func (hs *HTTPServ) Route(url string, h funcHandler) {
	handler := &httpHandler{fh: h}
	hs.routeMap[url] = handler
}

func (hs *HTTPServ) SetTimeout(t time.Duration)     { hs.timeout = t }
func (hs *HTTPServ) SetHeaders(h map[string]string) { hs.headers = h }
func (hs *HTTPServ) SetPort(p int)                  { hs.port = p }

// Start
func (hs *HTTPServ) Start() {
	mux := http.NewServeMux()
	for url, handler := range hs.routeMap {
		handler.headers = hs.headers
		mux.Handle(url, handler)
		utils.DebugLog("register route: ", url)
	}
	h := http.TimeoutHandler(mux, hs.timeout, "http time out!")

	utils.Log("Start Http Server...")
	http.ListenAndServe(fmt.Sprintf(":%d", hs.port), h)
}

type jsonResult struct {
	Errcode int         `json:"errcode"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// NewJson
func NewJson(data interface{}, errcode int, msg string) *jsonResult {
	return &jsonResult{
		Errcode: errcode,
		Data:    data,
		Message: msg,
	}
}

func (jr *jsonResult) ToBytes() []byte {
	b, err := json.Marshal(jr)
	if err != nil {
		utils.ErrorLog(err.Error())
		return NewErrorJson(1001, "failed marshal json").ToBytes()
	}
	return b
}

// NewErrorJson
func NewErrorJson(errcode int, msg string) *jsonResult {
	return &jsonResult{
		Errcode: errcode,
		Message: msg,
	}
}

// MyHTTPServ
type MyHTTPServ struct {
	routeMap map[string]*myHTTPHandler
	timeout  time.Duration
	port     string
}

// MyNewHTTPServ
func MyNewHTTPServ(port string) *MyHTTPServ {
	return &MyHTTPServ{
		routeMap: make(map[string]*myHTTPHandler),
		timeout:  30 * time.Second,
		port:     port,
	}
}

type myHTTPHandler struct {
	fh funcMyHandler
}

type funcMyHandler func(w http.ResponseWriter, request *http.Request)

func (hh *myHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, "+
		"X-Auth-Token, Authorization, Code, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	hh.fh(w, r)
}

// MyRoute
func (hs *MyHTTPServ) MyRoute(url string, h funcMyHandler) {
	handler := &myHTTPHandler{fh: h}
	hs.routeMap[url] = handler
}

// MyStart
func (hs *MyHTTPServ) MyStart() {
	mux := http.NewServeMux()
	for url, handler := range hs.routeMap {
		mux.Handle(url, handler)
		utils.DebugLog("register route: ", url)
	}
	h := http.TimeoutHandler(mux, hs.timeout, "http time out!")
	utils.Log("Start HTTP Server...")
	http.ListenAndServe(":"+hs.port, h)
}
