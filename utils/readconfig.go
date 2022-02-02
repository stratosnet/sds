package utils

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
		MyLogger.ErrorLog(err)
	}
	err = yaml.Unmarshal([]byte(data), &m)
	return m
}

// GetElement
func GetElement(key string, themap map[interface{}]interface{}) string {
	if value, ok := themap[key]; ok {

		return fmt.Sprint(value)
	}

	MyLogger.ErrorLog("can't find the config file")
	return ""
}

// LoadYamlConfig
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
