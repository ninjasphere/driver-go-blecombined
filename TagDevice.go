package main

import (
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/model"
)

type BLETag struct {
	driver          *BLETagDriver
	info            *model.Device
	sendEvent       func(event string, payload interface{}) error
	gattDevice      *gatt.DiscoveredDevice
	identifyChannel *channels.IdentifyChannel
	connected       bool
}

func NewBLETag(driver *BLETagDriver, gattDevice *gatt.DiscoveredDevice) error {

	if driver.FoundTags[gattDevice.Address] {
		// log.Debugf("Already found tag %s", gattDevice.Address)
		return nil
	} else {
		log.Debugf("Found BLE Tag %s", gattDevice.Address)
		name := "BLE Tag"

		bt := &BLETag{
			driver:     driver,
			gattDevice: gattDevice,
			connected:  false,
			info: &model.Device{
				NaturalID:     gattDevice.Address,
				NaturalIDType: "BLE Mac",
				Name:          &name, //TODO Fill me in with retrieved value
				Signatures: &map[string]string{
					"ninja:manufacturer": "Sticknfind",
					"ninja:productName":  "SL6",
					"ninja:productType":  "BLE Tag",
					"ninja:thingType":    "BLE Tag",
				},
			},
		}

		conn := driver.conn

		err := conn.ExportDevice(bt)
		if err != nil {
			btlog.Fatalf("Failed to export BLE Tag %+v %s", bt, err)
		}

		bt.identifyChannel = channels.NewIdentifyChannel(bt)
		err = conn.ExportChannel(bt, bt.identifyChannel, "identify")
		if err != nil {
			fplog.Fatalf("Failed to export BLE Tag identify channel %s, dumping device info", err)
			spew.Dump(bt)
		}

		gattDevice.Connected = bt.deviceConnected
		gattDevice.Disconnected = bt.deviceDisconnected

		driver.FoundTags[gattDevice.Address] = true

		return nil
	}
}

func (fp *BLETag) deviceConnected() {
	fp.connected = true
}

func (fp *BLETag) deviceDisconnected() {
	fp.connected = false
	log.Debugf("Disconnected from tag: %s", fp.gattDevice.Address)
}

func (fp *BLETag) GetDeviceInfo() *model.Device {
	return fp.info
}

func (fp *BLETag) GetDriver() ninja.Driver {
	return fp.driver
}

func (fp *BLETag) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	fp.sendEvent = sendEvent
}

func (fp *BLETag) Identify() error {
	for !fp.connected {
		log.Debugf("Connecting to tag %s", fp.gattDevice.Address)
		// spew.Dump(fp)
		err := client.Connect(fp.gattDevice.Address, fp.gattDevice.PublicAddress)
		if err != nil {
			log.Errorf("Connect error:%s", err)
			return err
		}
		time.Sleep(time.Second * 3) //call back on connect?
	}

	cmds := make([]string, 1)
	cmds[0] = "121b0002"
	client.SendRawCommands(fp.gattDevice.Address, cmds)
	time.Sleep(time.Second * 3)
	cmds[0] = "121b0000"
	client.SendRawCommands(fp.gattDevice.Address, cmds)
	return nil
}
