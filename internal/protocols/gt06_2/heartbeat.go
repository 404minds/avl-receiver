package gt06_2

import (
	"encoding/hex"
	"regexp"
)

type HeartbeatParser struct {
	Body   string
	Values []string
}

func NewHeartbeatParser(body string) *HeartbeatParser {
	return &HeartbeatParser{Body: body}
}

func (p *HeartbeatParser) IsValid() bool {
	re := regexp.MustCompile(`^(7878)([0-9a-f]{2})(13|23)`)
	p.Values = re.FindStringSubmatch(p.Body)
	return len(p.Values) > 0
}

func (p *HeartbeatParser) Response() string {
	if len(p.Values) == 0 {
		return ""
	}
	responseHex := p.Values[1] + "05" + p.Values[3] + "0001D9DC0D0A"
	responseBytes, _ := hex.DecodeString(responseHex)
	return string(responseBytes)
}
