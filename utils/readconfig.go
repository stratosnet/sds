package utils

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v2"
)

func GetYamlConfig(path string) map[interface{}]interface{} {
	data, err := ioutil.ReadFile(path)
	m := make(map[interface{}]interface{})
	if err != nil {
		MyLogger.ErrorLog(err)
	}
	err = yaml.Unmarshal([]byte(data), &m)
	return m
}

func GetElement(key string, themap map[interface{}]interface{}) string {
	if value, ok := themap[key]; ok {

		return fmt.Sprint(value)
	}

	MyLogger.ErrorLog("can't find the config file")
	return ""
}

func LoadYamlConfig(s interface{}, path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		MyLogger.ErrorLog(err)
		return err
	}
	if err = yaml.Unmarshal(file, s); err != nil {
		MyLogger.ErrorLog(err)
		return err
	}
	return nil
}

func WriteConfig(data interface{}, filePath string) error {
	yamlData, err := yaml.Marshal(&data)
	if err != nil {
		MyLogger.ErrorLog("Error while Marshaling.", err)
	}
	return ioutil.WriteFile(filePath, yamlData, 0644)
}

func LoadTomlConfig(s interface{}, path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		MyLogger.ErrorLog(err)
		return err
	}
	if err = toml.Unmarshal(file, s); err != nil {
		MyLogger.ErrorLog(err)
		return err
	}
	return nil
}

func WriteTomlConfig(data interface{}, filePath string) error {
	tomlData, err := toml.Marshal(data)
	if err != nil {
		MyLogger.ErrorLog("Error while Marshaling.", err)
	}
	return ioutil.WriteFile(filePath, tomlData, 0644)
}

func ParseTomlValue(data string) (interface{}, error) {
	if len(data) == 0 {
		return "", nil
	}
	if len(data) >= 2 && ((string(data[0]) == "\"" && string(data[len(data)-1]) == "\"") || (string(data[0]) == "'" && string(data[len(data)-1]) == "'")) {
		return data[1 : len(data)-1], nil
	}
	valInt, err := strconv.ParseInt(data, 10, 64)
	if err == nil {
		return valInt, nil
	}
	valFloat, err := strconv.ParseFloat(data, 64)
	if err == nil {
		return valFloat, nil
	}
	return strconv.ParseBool(data)
}
