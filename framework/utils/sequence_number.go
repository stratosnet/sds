package utils

import "fmt"

func GetSequenceNumberString(number uint64) string {
	return fmt.Sprintf("SN:%019d", number)
}
