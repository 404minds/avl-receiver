package gt06_2

import (
	"encoding/hex"
	"regexp"
)

type AuthParser struct {
	Body   string
	Values []string
}

func NewAuthParser(body string) *AuthParser {
	return &AuthParser{Body: body}
}

func (p *AuthParser) IsValid() bool {
	re := regexp.MustCompile(`^(7878)([0-9a-f]{2})(01)([0-9]{16})`)
	p.Values = re.FindStringSubmatch(p.Body)
	return len(p.Values) > 0
}

func (p *AuthParser) Response() string {
	if len(p.Values) == 0 {
		return ""
	}
	responseHex := p.Values[1] + "05" + p.Values[3] + "0001D9DC0D0A"
	responseBytes, _ := hex.DecodeString(responseHex)
	return string(responseBytes)
}
