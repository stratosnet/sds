package config

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/stratosnet/sds/utils"
)

// Connect config
type Connect struct {
	Dns     string `toml:"dns"`
	Driver  string `toml:"driver"`
	Host    string `toml:"host"`
	Port    uint16 `toml:"port"`
	User    string `toml:"user"`
	Pass    string `toml:"pass"`
	DbName  string `toml:"db_name"`
	Debug   bool   `toml:"debug"`
	LogFile string `toml:"log_file"`
}

// DNS
func (c *Connect) DNS() string {
	var dns string
	if c.Driver == "mysql" {
		if c.Dns == "" {
			dns = "$user:$pass@tcp($host:$port)/$dbName?charset=utf8"
			dns = strings.Replace(dns, "$user", c.User, -1)
			dns = strings.Replace(dns, "$pass", c.Pass, -1)
			dns = strings.Replace(dns, "$host", c.Host, -1)
			dns = strings.Replace(dns, "$port", strconv.FormatUint(uint64(c.Port), 10), -1)
			dns = strings.Replace(dns, "$dbName", c.DbName, -1)
		} else {
			dns = c.Dns
		}
	} else if c.Driver == "sqlite" {
		dns = c.Dns
	}
	return dns
}

// LoadConfFromYaml
func (c *Connect) LoadConfFromYaml(yamlFile string) {
	utils.LoadYamlConfig(c, yamlFile)
}

// LoadConfFromToml
func (c *Connect) LoadConfFromToml(tomlFile string) {
	utils.LoadTomlConfig(c, tomlFile)
}

// LoadConfFromMap
func (c *Connect) LoadConfFromMap(conf map[interface{}]interface{}) {

	config := make(map[string]interface{})
	for k, v := range conf {
		config[k.(string)] = v
	}

	fields := reflect.TypeOf(c).Elem()
	values := reflect.ValueOf(c)
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		index := utils.LcFirst(field.Name)
		value, ok := config[index]
		if ok {
			values.Elem().FieldByName(field.Name).Set(reflect.ValueOf(value).Convert(field.Type))
		}
	}
}
