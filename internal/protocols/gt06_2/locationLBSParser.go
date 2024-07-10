package gt06_2

import (
	"encoding/hex"
	"regexp"
	"strconv"
)

type LocationLbsParser struct {
	Body   string
	Values []string
	Cache  map[string]interface{}
}

func NewLocationLbsParser(body string) *LocationLbsParser {
	return &LocationLbsParser{Body: body, Cache: make(map[string]interface{})}
}

func (p *LocationLbsParser) IsValid() bool {
	re := regexp.MustCompile(`^(7878)([0-9a-f]{2})(12|22)([0-9a-f]{12})([0-9a-f]{2})([0-9a-f]{8})([0-9a-f]{8})([0-9a-f]{2})([0-9a-f]{4})([0-9a-f]{4})([0-9a-f]{2})`)
	p.Values = re.FindStringSubmatch(p.Body)
	return len(p.Values) > 0
}

func (p *LocationLbsParser) Latitude() float64 {
	latHex := p.Values[6]
	negative := p.Values[9][4] == '1'
	value := float64(int64(hex2Dec(latHex))) / 60 / 30000
	if negative {
		value *= -1
	}
	return value
}

func (p *LocationLbsParser) Longitude() float64 {
	lonHex := p.Values[7]
	negative := p.Values[9][5] == '0'
	value := float64(int64(hex2Dec(lonHex))) / 60 / 30000
	if negative {
		value *= -1
	}
	return value
}

func hex2Dec(hexStr string) int64 {
	value, _ := strconv.ParseInt(hexStr, 16, 64)
	return value
}

func (p *LocationLbsParser) Response() string {
	if len(p.Values) == 0 {
		return ""
	}
	responseHex := p.Values[1] + "05" + p.Values[3] + "0001D9DC0D0A"
	responseBytes, _ := hex.DecodeString(responseHex)
	return string(responseBytes)
}
