package teltonika

type TeltonikaRecord struct {
	IMEI   string             `json:"imei"`
	Record TeltonikaAvlRecord `json:"record"`
}

type TeltonikaAvlDataPacket struct {
	CodecID      uint8                `json:"codec_id"`
	NumberOfData uint8                `json:"number_of_data"`
	Data         []TeltonikaAvlRecord `json:"data"`
	CRC          uint32               `json:"crc"`
}

type TeltonikaAvlRecord struct {
	Timestamp  uint64              `json:"timestamp"`
	Priority   uint8               `json:"priority"`
	GPSElement TeltonikaGpsElement `json:"gps_element"`
	IOElement  TeltonikaIOElement  `json:"io_element"`
}

type TeltonikaGpsElement struct {
	Longitude  uint32 `json:"longitude"`
	Latitude   uint32 `json:"latitude"`
	Altitude   uint16 `json:"altitude"`
	Angle      uint16 `json:"angle"`
	Satellites uint8  `json:"satellites"`
	Speed      uint16 `json:"speed"`
}

type TeltonikaIOElement struct {
	EventID       uint8 `json:"event_id"`
	NumProperties uint8 `json:"num_properties"`

	Properties1B map[TeltonikaIOProperty]uint8  `json:"properties_1b"`
	Properties2B map[TeltonikaIOProperty]uint16 `json:"properties_2b"`
	Properties4B map[TeltonikaIOProperty]uint32 `json:"properties_4b"`
	Properties8B map[TeltonikaIOProperty]uint64 `json:"properties_8b"`
}

type TeltonikaIOProperty int

const (
	TIO_DigitalInput1      TeltonikaIOProperty = 1
	TIO_DigitalInput2                          = 2
	TIO_DigitalInput3                          = 3
	TIO_AnalogInput                            = 9
	TIO_PCBTemperature                         = 70
	TIO_DigitalOutput1                         = 179
	TIO_DigitalOutput2                         = 180
	TIO_GPSPDOP                                = 181
	TIO_GPSHDOP                                = 182
	TIO_ExternalVoltage                        = 66
	TIO_GPSPower                               = 69
	TIO_MovementSensor                         = 240
	TIO_OdometerValue                          = 199
	TIO_GSMOperator                            = 241
	TIO_Speed                                  = 24
	TIO_IButtonID                              = 78
	TIO_WorkingMode                            = 80
	TIO_GSMSignal                              = 21
	TIO_SleepMode                              = 200
	TIO_CellID                                 = 205
	TIO_AreaCode                               = 206
	TIO_DallasTemperature                      = 72
	TIO_BatteryVoltage                         = 67
	TIO_BatteryCurrent                         = 68
	TIO_AutoGeofence                           = 175
	TIO_Geozone1                               = 155
	TIO_Geozone2                               = 156
	TIO_Geozone3                               = 157
	TIO_Geozone4                               = 158
	TIO_Geozone5                               = 159
	TIO_TripMode                               = 250
	TIO_Immobilizer                            = 251
	TIO_AuthorizedDriving                      = 252
	TIO_GreenDrivingStatus                     = 253
	TIO_GreenDrivingValue                      = 254
	TIO_Overspeeding                           = 255
)

var iOProperties = []TeltonikaIOProperty{
	TIO_DigitalInput1, TIO_DigitalInput2, TIO_DigitalInput3,
	TIO_AnalogInput, TIO_PCBTemperature, TIO_DigitalOutput1,
	TIO_DigitalOutput2, TIO_GPSPDOP, TIO_GPSHDOP, TIO_ExternalVoltage,
	TIO_GPSPower, TIO_MovementSensor, TIO_OdometerValue, TIO_GSMOperator,
	TIO_Speed, TIO_IButtonID, TIO_WorkingMode, TIO_GSMSignal,
	TIO_SleepMode, TIO_CellID, TIO_AreaCode, TIO_DallasTemperature,
	TIO_BatteryVoltage, TIO_BatteryCurrent, TIO_AutoGeofence,
	TIO_Geozone1, TIO_Geozone2, TIO_Geozone3, TIO_Geozone4,
	TIO_Geozone5, TIO_TripMode, TIO_Immobilizer,
	TIO_AuthorizedDriving, TIO_GreenDrivingStatus, TIO_GreenDrivingValue,
	TIO_Overspeeding,
}

func IOPropertyFromID(id uint8) *TeltonikaIOProperty {
	for _, property := range iOProperties {
		if uint8(property) == id {
			return &property
		}
	}
	return nil
}
