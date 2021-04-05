package main

import (
	"fmt"
	"github.com/qsnetwork/sds/utils"
)

func main() {

	idWorker, err := utils.NewIdWorker(10)

	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(idWorker.NextIds(2))

}
