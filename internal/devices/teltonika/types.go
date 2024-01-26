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
	DigitalInput1      TeltonikaIOProperty = 1
	DigitalInput2                          = 2
	DigitalInput3                          = 3
	AnalogInput                            = 9
	PCBTemperature                         = 70
	DigitalOutput1                         = 179
	DigitalOutput2                         = 180
	GPSPDOP                                = 181
	GPSHDOP                                = 182
	ExternalVoltage                        = 66
	GPSPower                               = 69
	MovementSensor                         = 240
	OdometerValue                          = 199
	GSMOperator                            = 241
	Speed                                  = 24
	IButtonID                              = 78
	WorkingMode                            = 80
	GSMSignal                              = 21
	SleepMode                              = 200
	CellID                                 = 205
	AreaCode                               = 206
	DallasTemperature                      = 72
	BatteryVoltage                         = 67
	BatteryCurrent                         = 68
	AutoGeofence                           = 175
	Geozone1                               = 155
	Geozone2                               = 156
	Geozone3                               = 157
	Geozone4                               = 158
	Geozone5                               = 159
	TripMode                               = 250
	Immobilizer                            = 251
	AuthorizedDriving                      = 252
	GreenDrivingStatus                     = 253
	GreenDrivingValue                      = 254
	Overspeeding                           = 255
)

var iOProperties = []TeltonikaIOProperty{
	DigitalInput1, DigitalInput2, DigitalInput3, AnalogInput, PCBTemperature, DigitalOutput1, DigitalOutput2, GPSPDOP, GPSHDOP, ExternalVoltage, GPSPower, MovementSensor, OdometerValue, GSMOperator, Speed, IButtonID, WorkingMode, GSMSignal, SleepMode, CellID, AreaCode, DallasTemperature, BatteryVoltage, BatteryCurrent, AutoGeofence, Geozone1, Geozone2, Geozone3, Geozone4, Geozone5, TripMode, Immobilizer, AuthorizedDriving, GreenDrivingStatus, GreenDrivingValue, Overspeeding,
}

func IOPropertyFromID(id uint8) *TeltonikaIOProperty {
	for _, property := range iOProperties {
		if uint8(property) == id {
			return &property
		}
	}
	return nil
}
