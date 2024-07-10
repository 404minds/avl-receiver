package gt06_2

import (
	"encoding/hex"
	"regexp"
	"strconv"
)

type LocationGpsModularParser struct {
	Body   string
	Values []string
	Cache  map[string]interface{}
}

func NewLocationGpsModularParser(body string) *LocationGpsModularParser {
	return &LocationGpsModularParser{Body: body, Cache: make(map[string]interface{})}
}

func (p *LocationGpsModularParser) IsValid() bool {
	re := regexp.MustCompile(`^(7979)([0-9a-f]{4})(70)`)
	p.Values = re.FindStringSubmatch(p.Body)
	return len(p.Values) > 0
}

func (p *LocationGpsModularParser) Modules() {
	buffer := p.Body[10 : len(p.Body)-8]
	for len(buffer) > 6 {
		moduleType := buffer[:2]
		contentLength, _ := strconv.ParseInt(buffer[2:4], 16, 64)
		content := buffer[4 : 4+contentLength*2]
		switch moduleType {
		case "0011":
			// Handle cell tower module
		case "0033":
			// Handle GPS module
		case "002C":
			// Handle timestamp module
		default:
		}
		buffer = buffer[4+contentLength*2:]
	}
}

func (p *LocationGpsModularParser) Response() string {
	if len(p.Values) == 0 {
		return ""
	}
	responseHex := p.Values[1] + "0005" + p.Values[3] + "0001D9DC0D0A"
	responseBytes, _ := hex.DecodeString(responseHex)
	return string(responseBytes)
}
