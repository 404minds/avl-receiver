package crc

import "github.com/snksoft/crc"

func Crc_Teltonika(data []byte) uint16 {
	checksum := crc.CalculateCRC(&crc.Parameters{Width: 16, Polynomial: 0x8005, ReflectIn: true, ReflectOut: true, Init: 0x0000, FinalXor: 0x0000}, data)
	return uint16(checksum)
}

func Crc_Wanway(data []byte) uint16 {
	checksum := crc.CalculateCRC(crc.X25, data)
	return uint16(checksum)
}
