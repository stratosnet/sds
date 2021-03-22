package cache

import (
	"time"
)

// Redis Config
type Config struct {
	Engine   string `yaml:"Engine"`
	Host     string `yaml:"Host"`
	Port     string `yaml:"Port"`
	Pass     string `yaml:"Pass"`
	DB       int    `yaml:"DB"`
	LifeTime int64  `yaml:"LifeTime"`
}

// Cache
type Cache interface {
	IsOK() error
	Get(key string, data interface{}) error
	Delete(key string) error
	Set(key string, value interface{}, expire time.Duration) error
	EnQueue(key string, value interface{}) error
	DeQueue(key string) (interface{}, error)
}
