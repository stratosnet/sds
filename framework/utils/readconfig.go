package utils

import (
	"os"
	"strconv"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v2"
)

func LoadYamlConfig(s interface{}, path string) error {
	file, err := os.ReadFile(path)
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

func LoadTomlConfig(s interface{}, path string) error {
	file, err := os.ReadFile(path)
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
	return os.WriteFile(filePath, tomlData, 0644)
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
