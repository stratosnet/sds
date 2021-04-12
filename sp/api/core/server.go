package core

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/stratosnet/sds/sp/storages"
	"github.com/stratosnet/sds/sp/tools"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database"
	"github.com/stratosnet/sds/utils/database/config"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Config
type Config struct {
	Version  uint32         `yaml:"Version"`
	Host     string         `yaml:"Host"`
	Port     int            `yaml:"Port"`
	Cache    cache.Config   `yaml:"Cache"`
	Database config.Connect `yaml:"Database"`
	LogFile  string         `yaml:"LogFile"`
	Debug    bool           `yaml:"Debug"`
}

// Handler
type Handler interface {
	SetAPIServer(server *APIServer)
	GetAPIServer() *APIServer
}

// APIServer
type APIServer struct {
	Host      string
	Port      int
	server    *mux.Router
	Conf      *Config
	DB        *database.DataTable
	Logger    *utils.Logger
	FreshTime time.Duration
	storages.ServerCache
}

// AddHandler
func (s *APIServer) AddHandler(method string, pattern string, handler Handler, handleName string) {

	if s.server == nil {
		s.Logger.Log(utils.Error, "server is not init")
		return
	}

	handler.SetAPIServer(s)

	s.server.HandleFunc(pattern, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("content-type", "application/json")
		var err error
		params := make(map[string]interface{})
		responseJSON := new(tools.JsonResult)

		val := reflect.ValueOf(handler)
		params, err = s.ParseBody(request)
		if err != nil {
			responseJSON.Errcode = 500
			responseJSON.Message = err.Error()
		} else {
			handleResult := val.MethodByName(handleName).Call([]reflect.Value{reflect.ValueOf(params), reflect.ValueOf(request)})
			if len(handleResult) == 3 {
				responseJSON = tools.NewJson(handleResult[0].Interface(), handleResult[1].Interface().(int), handleResult[2].String())
			}
		}
		writer.Write(responseJSON.ToBytes())

		if s.Conf.Debug {
			paramsJSON, _ := json.Marshal(params)
			s.Log(request.Method, request.RequestURI, string(paramsJSON), "=>", string(responseJSON.ToBytes()))
		}
	}).Methods(method)
}

// Start 启动服务
func (s *APIServer) Start() {

	s.server.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("sp/web/panel"))))
	s.server.PathPrefix("/statics").Handler(http.StripPrefix("/statics", http.FileServer(http.Dir("sp/web/panel/statics"))))
	http.ListenAndServe(s.Host+":"+strconv.Itoa(s.Port), s.server)
}

// ParseBody 解析参数
func (s *APIServer) ParseBody(r *http.Request) (map[string]interface{}, error) {

	data := make(map[string]interface{})

	log := strings.Join([]string{r.Method, r.RequestURI}, " ")

	switch r.Header.Get("Content-Type") {

	case "application/json":
		body, _ := ioutil.ReadAll(r.Body)
		if len(body) > 0 {
			log = log + " " + string(body)
			err := json.Unmarshal(body, &data)
			if err != nil {
				return nil, errors.New("参数解析错误")
			}
			r.Body.Close()
		}
	}

	return data, nil
}

// Log 记录日志
func (s *APIServer) Log(log ...interface{}) {
	if s.Logger == nil {
		utils.Log(log...)
	} else {
		s.Logger.Log(utils.Info, log...)
	}
}

// NewApiServer 新建一个Api服务
func NewAPIServer(config *Config) *APIServer {

	api := new(APIServer)
	api.Conf = config
	api.Host = config.Host
	api.Port = config.Port

	api.server = mux.NewRouter()
	api.DB = database.NewDataTable(config.Database)
	api.Cache = cache.NewRedis(config.Cache)

	if config.LogFile != "" {

		path, _ := filepath.Abs(filepath.Dir(config.LogFile))
		if _, err := os.Stat(path); err != nil {
			err := os.MkdirAll(path, 0711)
			if err != nil {
				utils.ErrorLog("creating directory failed")
				return nil
			}
		}

		api.Logger = utils.NewLogger(config.LogFile, true, true)
		api.Logger.SetLogLevel(utils.Info)
	}

	return api
}
