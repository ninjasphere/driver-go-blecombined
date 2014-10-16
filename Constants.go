package main

import "time"

const (
	flowerPowerServiceUuid = "39e1fa0084a811e2afba0002a5d5c51b"
	liveModeUuid           = "39e1fa0684a811e2afba0002a5d5c51b"
	sunlightHandle         = 37
	temperatureHandle      = 49
	moistureHandle         = 53
	batteryHandle          = 68
	sunlightStartHandle    = 36
	sunlightEndHandle      = 39
	moistureStartHandle    = 52
	moistureEndHandle      = 55
	temperatureStartHandle = 48
	temperatureEndHandle   = 51
	waypointStartHandle    = 45
	waypointEndHandle      = 48
	dataInterval           = time.Second * 5
	sleepInterval          = dataInterval
	// sleepInterval          = time.Minute * 30
)
