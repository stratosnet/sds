package main

import (
	"github.com/stratosnet/sds/sp"
	"github.com/stratosnet/sds/utils"
)

func main() {
	utils.NewLogger("stdout.log", true, true)
	sp.StartSP("configs/sp.yaml")
}
