package cache

import (
	"time"
)

const (
	PREFIX = "#"
	KEY_PP = "PP"
)

var (
	TtlPP = time.Second * time.Duration(600)
)
