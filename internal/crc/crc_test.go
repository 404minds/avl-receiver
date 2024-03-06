package crc

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWanwayCrc(t *testing.T) {
	testCases := []struct {
		input    string
		expected uint16
	}{
		{"11 01 07 52 53 36 78 90 02 42 70 00 32 01 00 05", 0x1279},
		{"0A 13 40 04 04 00 01 00 0F", 0xDCEE},
		{"22 22 0F 0C 1D 02 33 05 C9 02 7A C8 18 0C 46 58 60 00 14 00 01 CC 00 28 7D 00 1F 71 00 00 01 00 08", 0x2086},
		{"3B 28 10 01 0D 02 02 02 01 CC 00 28 7D 00 1F 71 3E 28 7D 00 1F 72 31 28 7D 00" +
			"1E 23 2D 28 7D 00 1F 40 18 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF 00 02 00" +
			"05", 0xB14B},
	}

	for _, testcase := range testCases {
		data, _ := hex.DecodeString(strings.ReplaceAll(testcase.input, " ", ""))
		crc := Crc_Wanway(data)
		assert.Equal(t, testcase.expected, crc, "crc should match")
	}
}

func TestTeltonikaCrc(t *testing.T) {
	testCases := []struct {
		input    string
		expected uint16
	}{
		{"08030000013FEB40E0B2000F0EC760209A6B000062000006000000170" +
		"A010002000300B300B4004501F00150041503C80008B50012B6000A423024180000CD0386" +
		"CE0001431057440000044600000112C700000000F10000601A4800000000014E000000000" +
		"00000000000013F14A1D1CE000F0EB790209A778000AB010C050000000000000000000001" +
		"3F1498A63A000F0EB790209A77800095010C04000000000000000003", 0x3390},
	}

	for _, testcase := range testCases {
		data, _ := hex.DecodeString(testcase.input)
		crc := Crc_Teltonika(data)
		assert.Equal(t, testcase.expected, crc, "crc should match")
	}
}
