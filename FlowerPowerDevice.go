package main

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"math"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/model"
)

type FlowerPower struct {
	driver             *FlowerPowerDriver
	info               *model.Device
	sendEvent          func(event string, payload interface{}) error
	gattDevice         *gatt.DiscoveredDevice
	temperatureChannel *channels.TemperatureChannel
	moistureChannel    *channels.MoistureChannel
	illuminanceChannel *channels.IlluminanceChannel
	connected          bool
}

func NewFlowerPower(driver *FlowerPowerDriver, gattDevice *gatt.DiscoveredDevice) error {

	name := "FlowerPower"

	fp := &FlowerPower{
		driver:     driver,
		gattDevice: gattDevice,
		connected:  false,
		info: &model.Device{
			NaturalID:     gattDevice.Address,
			NaturalIDType: "FlowerPower",
			Name:          &name, //TODO Fill me in with retrieved value
			Signatures: &map[string]string{
				"ninja:manufacturer": "Parrot",
				"ninja:productName":  "FlowerPower",
				"ninja:productType":  "FlowerPower",
				"ninja:thingType":    "plant sensor",
			},
		},
	}

	conn := driver.conn

	fp.temperatureChannel = channels.NewTemperatureChannel(fp)
	err := conn.ExportChannel(fp, fp.temperatureChannel, "temperature")
	if err != nil {
		fplog.Fatalf("Failed to export flowerpower temperature channel %s, dumping device info", err)
		spew.Dump(fp)
	}

	fp.moistureChannel = channels.NewMoistureChannel(fp)
	err = conn.ExportChannel(fp, fp.moistureChannel, "moisture")
	if err != nil {
		fplog.Fatalf("Failed to export flowerpower moisture channel %s, dumping device info", fp, err)
		spew.Dump(fp)
	}

	fp.illuminanceChannel = channels.NewIlluminanceChannel(fp)
	err = conn.ExportChannel(fp, fp.illuminanceChannel, "illuminance")
	if err != nil {
		fplog.Fatalf("Failed to export flowerpower illuminance channel %s, dumping device info", err)
		spew.Dump(fp)
	}

	gattDevice.Connected = fp.deviceConnected
	gattDevice.Disconnected = fp.deviceDisconnected
	gattDevice.Notification = fp.handleFPNotification

	err = conn.ExportDevice(fp)
	if err != nil {
		fplog.Fatalf("Failed to export flowerpower %+v %s", fp, err)
	}

	fp.startFPLoop(gattDevice)

	fp.driver.announcedFlowerPowers[gattDevice.Address] = true
	return nil
}

func (fp *FlowerPower) startFPLoop(gattDevice *gatt.DiscoveredDevice) {
	go func() {
		for {

			if fp.driver.running == true {

				if fp.connected == false {
					fplog.Debugf("Connecting to Flower Power %s", gattDevice.Address)
					err := fp.driver.gattClient.Connect(gattDevice.Address, gattDevice.PublicAddress)
					if err != nil {
						fplog.Errorf("Flowerpower connect error:%s", err)
					}
					time.Sleep(time.Second * 5) //sorry :(
				}

				if fp.connected == true {
					fplog.Debugf("Connected to flower power: %s", fp.gattDevice.Address)
					fplog.Debugf("Setting up notifications")
					fp.notifyAll()
					fplog.Debugf("Enabling live mode")
					fp.EnableLiveMode()
					time.Sleep(dataInterval)
					fplog.Debugf("Disabling live mode")
					fp.DisableLiveMode()
					time.Sleep(sleepInterval)
				}
			}
		}
	}()
}

func (fp *FlowerPower) handleFPNotification(notification *gatt.Notification) {
	if notification.Handle == sunlightHandle {
		sunlight := parseSunlight(notification.Data)
		fplog.Debugf("Got sunlight: %f", sunlight)
		fp.illuminanceChannel.SendState(sunlight)

	} else if notification.Handle == moistureHandle {
		moisture := parseMoisture(notification.Data)
		fplog.Debugf("Got moisture: %f", moisture)
		fp.moistureChannel.SendState(moisture)

	} else if notification.Handle == temperatureHandle {
		temperature := parseTemperature(notification.Data)
		fplog.Debugf("Got temperature: %f", temperature)
		fp.temperatureChannel.SendState(temperature)

	} else {
		fplog.Infof("Unknown notification handle")
		spew.Dump(notification)
	}
}

func (fp *FlowerPower) notifyAll() {
	fp.notifyByHandle(sunlightStartHandle, sunlightEndHandle)
	fp.notifyByHandle(moistureStartHandle, moistureEndHandle)
	fp.notifyByHandle(temperatureStartHandle, temperatureEndHandle)
}

func (fp *FlowerPower) deviceConnected() {
	fp.connected = true
}

func (fp *FlowerPower) deviceDisconnected() {
	fp.connected = false
}

func (fp *FlowerPower) GetDeviceInfo() *model.Device {
	return fp.info
}

func (fp *FlowerPower) GetDriver() ninja.Driver {
	return fp.driver
}

func (fp *FlowerPower) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	fp.sendEvent = sendEvent
}

//TOOD: this needs to send via a proper handle and value, rather than sending a raw packet
func (fp *FlowerPower) EnableLiveMode() {
	cmds := make([]string, 1)
	cmds[0] = "12390001"
	fp.driver.gattClient.SendRawCommands(fp.gattDevice.Address, cmds)

}

//TOOD: this needs to send via a proper handle and value, rather than sending a raw packet
func (fp *FlowerPower) DisableLiveMode() {
	cmds := make([]string, 1)
	cmds[0] = "12390000"
	fp.driver.gattClient.SendRawCommands(fp.gattDevice.Address, cmds)
}

func (fp *FlowerPower) notifyByHandle(startHandle, endHandle uint16) {
	fp.driver.gattClient.Notify(fp.gattDevice.Address, true, startHandle, endHandle, true, false)
}

func parseSunlight(data []byte) float64 {
	sensorVal := bytesToUint(data)
	if sensorVal < 0 {
		sensorVal = 0
	} else if sensorVal > 65530 {
		sensorVal = 65530
	}
	rounded := math.Floor(float64(sensorVal)/10) * 10 //Only 10% of data in mapping

	return getValFromMap("sunlight.json", rounded)
}

func parseMoisture(data []byte) float64 {
	sensorVal := bytesToUint(data)
	if sensorVal < 210 {
		sensorVal = 210
	} else if sensorVal > 700 {
		sensorVal = 700
	}
	return getValFromMap("soil-moisture.json", float64(sensorVal))
}

func parseTemperature(data []byte) float64 {
	sensorVal := bytesToUint(data)
	if sensorVal < 210 {
		sensorVal = 210
	} else if sensorVal > 1372 {
		sensorVal = 1372
	}
	return getValFromMap("temperature.json", float64(sensorVal))
}

func (fp *FlowerPower) getValFromHandle(handle int) []byte {
	// log.Debugf("--Readbyhandle-- address: %s handle: %d", fp.gattDevice.Address, handle)
	data := <-fp.driver.gattClient.ReadByHandle(fp.gattDevice.Address, uint16(handle))
	return data
}

func getValFromMap(filename string, sensorVal float64) float64 {
	mapFile, err := ioutil.ReadFile("data/" + filename)
	if err != nil {
		fplog.Fatalf("Error reading %s json map file: %s", filename, err)
	}

	mapJson, err := simplejson.NewFromReader(bytes.NewBuffer(mapFile))
	if err != nil {
		fplog.Fatalf("Error creating reader: %s", err)
	}

	sensorValStr := strconv.Itoa(int(sensorVal))

	ret, err := mapJson.Get(sensorValStr).Float64()

	if err != nil {
		fplog.Infof("Error parsing sensor value %f with stringified value %s to mapped value: %s", sensorVal, sensorValStr, err)
	}

	return ret
}

func bytesToUint(in []byte) uint16 {
	var ret uint16
	buf := bytes.NewReader(in)
	err := binary.Read(buf, binary.LittleEndian, &ret)
	if err != nil {
		fplog.Errorf("bytesToUint: Couldn't convert bytes % X to uint", in)
		return 0
	}
	return ret
}

func (fp *FlowerPower) GetSunlight() float64 {
	sensorVal := fp.getValFromHandle(sunlightHandle)
	return parseSunlight(sensorVal)
}

func (fp *FlowerPower) GetTemperature() float64 {
	sensorVal := fp.getValFromHandle(temperatureHandle)
	return parseTemperature(sensorVal)
}

func (fp *FlowerPower) GetMoisture() float64 {
	sensorVal := fp.getValFromHandle(moistureHandle)
	return parseMoisture(sensorVal)
}

func (fp *FlowerPower) GetBatteryLevel() float64 {
	value := bytesToUint(fp.getValFromHandle(batteryHandle))
	return float64(value)
}
