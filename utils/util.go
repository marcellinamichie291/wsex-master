/*
@Time : 2020/6/10 5:08 下午
@Author : shiguantian
@File : util.go
@Software: GoLand
*/
package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const ZERO float64 = 0.0000000001

func Round(f float64, n int, rounding bool) string {
	n10 := math.Pow10(n)
	add := 0.0
	if rounding {
		// 是否四舍五入
		add = 0.5 / n10
	}
	return fmt.Sprintf("%."+strconv.Itoa(n)+"f", math.Trunc((f+add)*n10)/n10)
}

func SafeAssign(src interface{}, dst interface{}) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return
	}

	if reflect.TypeOf(src).Kind() == v.Elem().Kind() && reflect.TypeOf(src).String() == v.Elem().Type().String() {
		v.Elem().Set(reflect.ValueOf(src))
	}
}

func SafeMapAssign(data map[string]interface{}, key string, dst interface{}) {
	src, ok := data[key]
	if !ok {
		return
	}

	SafeAssign(src, dst)
}

func SafeParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

const (
	CompareInvalid = iota
	CompareEqual
	CompareLess
	CompareGreater
)

func CompareFloatString(left, right string) int {
	fL, err1 := strconv.ParseFloat(left, 64)
	fR, err2 := strconv.ParseFloat(right, 64)
	if err1 != nil || err2 != nil {
		return CompareInvalid
	}
	if math.Abs(fL-fR) < ZERO {
		return CompareEqual
	} else if fL > fR {
		return CompareGreater
	} else {
		return CompareLess
	}
}

func EpochTime() string {
	millisecond := time.Now().UnixNano() / 1000000
	epoch := strconv.Itoa(int(millisecond))
	epochBytes := []byte(epoch)
	epoch = string(epochBytes[:10]) + "." + string(epochBytes[10:])
	return epoch
}

func IsoTime() string {
	utcTime := time.Now().UTC()
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:23]) + "Z"
	return iso
}

func ParseIsoTime(timestamp string, loc *time.Location) time.Duration {
	if loc == nil {
		loc, _ = time.LoadLocation("")
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05Z", timestamp, loc)
	if err != nil {
		return 0
	}
	t = t.Local()
	return time.Duration(t.UnixNano()) / 1e6
}

func GenerateOrderClientId(prefix string, size int) string {
	uuidStr := strings.Replace(uuid.New().String(), "-", "", 32)
	if prefix == "" {
		prefix = "wsex"
	}
	return prefix + uuidStr[0:size-len(prefix)]
}

func IsClientOrderID(orderID, prefix string) bool {
	if prefix == "" {
		prefix = "wsex"
	}
	return strings.Contains(orderID, prefix)
}

func UrlValuesToJson(values url.Values) string {
	m := make(map[string]interface{}, 0)
	for key, val := range values {
		if len(val) > 1 {
			m[key] = val
		} else if len(val) == 1 {
			m[key] = val[0]
		}
	}
	js, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(js)
}
