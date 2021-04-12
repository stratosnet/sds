package main

import (
	"fmt"
	"github.com/stratosnet/sds/sp/storages"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/utils/cache"
)

func main() {

	c := storages.NewServerCache(cache.NewRedis(cache.Config{
		Engine: "redis",
		Host:   "localhost",
		Port:   "6789",
		Pass:   "123456",
		DB:     0,
	}))

	sys := &data.System{
		Version: 1,
	}

	c.Store(sys, 0)

	state2 := new(data.System)

	state2.Version = 2

	c.Load(state2)

	fmt.Println(state2)
}
