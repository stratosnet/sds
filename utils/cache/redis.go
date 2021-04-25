package cache

import (
	"encoding/json"
	"errors"
	"github.com/go-redis/redis"
	"github.com/stratosnet/sds/utils"
	"time"
)

// Redis client
type Redis struct {
	Client *redis.Client
}

// IsOK pings the redis server and check server's pong response
func (r *Redis) IsOK() error {

	if pong, err := r.Client.Ping().Result(); err == nil && pong == "PONG" {
		return nil
	}

	return errors.New("redis can not ping")
}

// Get
func (r *Redis) Get(key string, data interface{}) error {

	if key == "" {
		return errors.New("key is nil")
	}
	res, err := r.Client.Get(key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(res), data)
}

// Set
func (r *Redis) Set(key string, value interface{}, expire time.Duration) error {

	if value != nil && key != "" {
		data, err := json.Marshal(value)
		if err != nil {
			utils.Log(err)
			return err
		}
		return r.Client.Set(key, data, expire).Err()
	}

	return errors.New("key or value is nil")
}

// Delete
func (r *Redis) Delete(key string) error {

	if key != "" {
		res, err := r.Client.Del(key).Result()
		if err == nil && res >= 0 {
			return nil
		}
		utils.Log(err)
	}

	return errors.New("key is nil")
}

// EnQueue
func (r *Redis) EnQueue(key string, value interface{}) error {
	return r.Client.LPush(key, value).Err()
}

// DeQueue
func (r *Redis) DeQueue(key string) (interface{}, error) {
	return r.Client.RPop(key).Result()
}

// NewRedis 实例化一个redis缓存
func NewRedis(config Config) *Redis {

	r := &Redis{
		Client: redis.NewClient(&redis.Options{
			Addr:     config.Host + ":" + config.Port,
			Password: config.Pass,
			DB:       config.DB,
		}),
	}

	if err := r.IsOK(); err != nil {
		utils.Log(err)
	}

	return r
}
