package main

import (
	"fmt"
	"github.com/qsnetwork/sds/utils"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func main() {

	fmt.Println(utils.CalcHash([]byte(uuid.New().String() + "#" + strconv.FormatInt(time.Now().UnixNano(), 10))))

	fmt.Println(utils.Get8BitUUID())

}
