package cache

import (
	"time"
)

// Redis Config
type Config struct {
	Engine   string `toml:"engine"`
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	Pass     string `toml:"pass"`
	DB       int    `toml:"db"`
	LifeTime int64  `toml:"life_time"`
}

// Cache
type Cache interface {
	IsOK() error
	Get(key string, data interface{}) error
	Delete(key string) error
	Set(key string, value interface{}, expire time.Duration) error
	Expire(key string, expire time.Duration) error
	EnQueue(key string, value interface{}) error
	DeQueue(key string) (interface{}, error)
}
