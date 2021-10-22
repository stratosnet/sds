package utils

import (
	"bytes"
	"crypto/md5"
	cryptorand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"github.com/google/uuid"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	var seed [8]byte
	_, err := cryptorand.Read(seed[:])
	if err != nil {
		panic("cannot set random seed from cryptographically secure random number")
	}
	rand.Seed(int64(binary.LittleEndian.Uint64(seed[:])))
}

// Camel2Snake
// eg. HelloWorld => hello_world
func Camel2Snake(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

// Snake2Camel
// eg. hello_world => HelloWorld
func Snake2Camel(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

// UcFirst first letter upper case
func UcFirst(str string) string {
	tmp := []rune(str)
	tmpLen := len(tmp)
	s := make([]rune, 0, tmpLen)
	if tmpLen > 0 {
		s = append(s, tmp[0]-32)
		s = append(s, tmp[1:]...)
		return string(s)
	}
	return ""
}

// LcFirst first letter lower case
func LcFirst(str string) string {
	tmp := []rune(str)
	tmpLen := len(tmp)
	s := make([]rune, 0, tmpLen)
	if tmpLen > 0 {
		s = append(s, tmp[0]+32)
		s = append(s, tmp[1:]...)
		return string(s)
	}
	return ""
}

// ConvertCoronaryUtf8
func ConvertCoronaryUtf8(in string) string {
	s := []byte(in)
	reg := regexp.MustCompile(`\\[0-7]{3}`)

	out := reg.ReplaceAllFunc(s,
		func(b []byte) []byte {
			i, _ := strconv.ParseInt(string(b[1:]), 8, 0)
			return []byte{byte(i)}
		})
	return string(out)
}

// StrInSlices
func StrInSlices(slices []string, key string) bool {
	if len(slices) > 0 {
		for _, k := range slices {
			if k == key {
				return true
			}
		}
	}
	return false
}

// GenerateRandomNumber generate (count) random numbers between (start) to (end).
func GenerateRandomNumber(start int, end int, count int) []int {
	if end < start || (end-start) < count {
		return nil
	}

	// output numbers
	nums := make([]int, 0)
	for len(nums) < count {

		num := rand.Intn(end-start) + start

		// find duplicates
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}

		if !exist {
			nums = append(nums, num)
		}
	}

	return nums
}

// GetRandomString between [0-9a-zA-Z]
func GetRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}

	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func GetMD5(data string) string {
	if data != "" {
		md5Obj := md5.New()
		md5Obj.Write([]byte(data))
		return hex.EncodeToString(md5Obj.Sum(nil))
	}
	return ""
}

func Get16MD5(data string) string {
	md5InStr := GetMD5(data)
	if md5InStr != "" {
		md5InByte := []byte(md5InStr)
		if len(md5InByte) == 32 {
			var Bytes bytes.Buffer
			Bytes.Write(md5InByte[8:24])
			return Bytes.String()
		}
	}
	return ""
}

// Get8BitUUID
func Get8BitUUID() string {
	chars := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	}
	var buff bytes.Buffer
	uuidInString := uuid.New().String()
	for i := 0; i < 8; i++ {
		str := []byte(uuidInString[i*4 : i*4+4])
		buff.WriteString(chars[CalcCRC32(str)%0x3E])
	}
	return buff.String()
}
