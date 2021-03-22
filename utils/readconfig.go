package utils

// Author cc
import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// GetYamlConfig
func GetYamlConfig(path string) map[interface{}]interface{} {
	data, err := ioutil.ReadFile(path)
	m := make(map[interface{}]interface{})
	if err != nil {
		fmt.Println(err)
	}
	err = yaml.Unmarshal([]byte(data), &m)
	return m
}

// GetElement
func GetElement(key string, themap map[interface{}]interface{}) string {
	if value, ok := themap[key]; ok {

		return fmt.Sprint(value)
	}

	fmt.Println("can't find the config file")
	return ""
}

// LoadYamlConfig
func LoadYamlConfig(s interface{}, path string) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		MyLogger.ErrorLog(err)
	}
	if err = yaml.Unmarshal(file, s); err != nil {
		MyLogger.ErrorLog(err)
	}
}
