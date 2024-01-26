package crc

import "github.com/snksoft/crc"

func Crc16_IBM(data []byte) uint16 {
	checksum := crc.CalculateCRC(&crc.Parameters{Width: 16, Polynomial: 0x8005, ReflectIn: true, ReflectOut: true, Init: 0x0000, FinalXor: 0x0000}, data)
	return uint16(checksum)
}
