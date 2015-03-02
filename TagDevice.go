package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/driver-go-blecombined/bluez"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/model"
)

// const stickNFindServiceUUID = "cd54cc79ce6c4cf49747447e0fbe6295"
// const stickNFindReadUUID = "8da7135268044fc0b8dd34a5389ed0d0"
// const stickNFindWriteUUID = "673e2b625b2c4bb0876d87bcaa06d66f"

const (
	stickNFindReadUUID  = "8da71352-6804-4fc0-b8dd-34a5389ed0d0"
	stickNFindWriteUUID = "673e2b62-5b2c-4bb0-876d-87bcaa06d66f"
)

type BLETag struct {
	sync.Mutex
	driver          *BLETagDriver
	info            *model.Device
	sendEvent       func(event string, payload interface{}) error
	address         string
	identifyChannel *channels.IdentifyChannel
	onOffChannel    *channels.OnOffChannel

	// currently we are using the bluez gatttool wrapper due to issues with
	// access characteristics with security enabled.
	gattCmd   *bluez.GattCmd
	readChar  *bluez.Characteristic
	alertChar *bluez.Characteristic

	device *gatt.DiscoveredDevice

	// service   gatt.ServiceDescription
	// readChar  gatt.CharacteristicDescription
	// writeChar gatt.CharacteristicDescription
	// alertChar gatt.CharacteristicDescription
}

func NewBLETag(driver *BLETagDriver, device *gatt.DiscoveredDevice) error {

	address := device.Address

	if driver.FoundTags[address] {
		log.Infof("Already found tag %s", address)
		return nil
	}

	log.Infof("Found BLE Tag address=%s public=%v", address, device.PublicAddress)

	name := "BLE Tag"

	bt := &BLETag{
		driver: driver,
		device: device,
		info: &model.Device{
			NaturalID:     address,
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

	bt.onOffChannel = channels.NewOnOffChannel(bt)
	err = conn.ExportChannel(bt, bt.onOffChannel, "on-off")
	if err != nil {
		fplog.Fatalf("Failed to export BLE Tag on-off channel %s, dumping device info", err)
		spew.Dump(bt)
	}

	driver.FoundTags[address] = true

	if device.PublicAddress {
		return fmt.Errorf("Public addresses not supported, upgrade the firmware!")
	}

	bt.gattCmd = bluez.NewGattCmd(address, bluez.AddrTypeRandom)

	chars, err := bt.gattCmd.ReadCharacteristics()

	if err != nil {
		return fmt.Errorf("Discovery Error: %s", err)
	}

	for _, char := range chars {
		if char.UUID == stickNFindReadUUID {
			bt.readChar = char
		}
		if char.UUID == stickNFindWriteUUID {
			bt.alertChar = char
		}
	}

	if bt.alertChar == nil {
		return fmt.Errorf("Alert characteristic not found")
	}

	if bt.readChar == nil {
		return fmt.Errorf("Read characteristic not found")
	}

	time.Sleep(1 * time.Second)

	if err := bt.gattCmd.WriteCharacteristic(bt.alertChar.CharValueHandle, "0103"); err != nil {
		return fmt.Errorf("Alert characteristic write failed: %v", err)
	}

	return nil
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

func (fp *BLETag) ReadStatus() []byte {

	data, err := fp.gattCmd.ReadCharacteristic(fp.readChar.CharValueHandle)

	if err != nil {
		log.Errorf("ReadStatus failed: %s", err)
	}

	return data
}

// Only temporary! This shouldn't be an on-off device.
// We ignore the state....
func (fp *BLETag) SetOnOff(_ bool) error {
	state, err := fp.Buzz()

	select {
	case <-state:
		log.Infof("Started buzzing")
		fp.onOffChannel.SendEvent("state", true)
		<-state
		log.Infof("Stopped buzzing")
		fp.onOffChannel.SendEvent("state", false)
	case e := <-err:
		log.Warningf("Failed to buzz", e)
		return e
	}
	return nil
}

func (fp *BLETag) ToggleOnOff() error {
	return fp.SetOnOff(true)
}

func (fp *BLETag) Buzz() (state chan bool, errChan chan error) {

	fp.Lock()

	state, errChan = make(chan bool, 2), make(chan error, 1)

	go func() {

		defer fp.Unlock()

		if !fp.driver.running {
			errChan <- fmt.Errorf("Driver not running, but received identify command")
			return
		}

		if err := fp.gattCmd.WriteCharacteristic(fp.alertChar.CharValueHandle, "0103"); err != nil {
			errChan <- fmt.Errorf("Alert characteristic write failed: %q", err)
		}

	}()

	return
}

func (fp *BLETag) Identify() error {
	state, err := fp.Buzz()

	select {
	case <-state:
		log.Infof("Started identifying")
	case e := <-err:
		log.Warningf("Failed to identify", e)
		return e
	}
	return nil
}
