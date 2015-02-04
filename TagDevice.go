package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/model"
)

const stickNFindServiceUUID = "cd54cc79ce6c4cf49747447e0fbe6295"
const stickNFindReadUUID = "8da7135268044fc0b8dd34a5389ed0d0"
const stickNFindWriteUUID = "673e2b625b2c4bb0876d87bcaa06d66f"

type BLETag struct {
	sync.Mutex
	driver          *BLETagDriver
	info            *model.Device
	sendEvent       func(event string, payload interface{}) error
	address         string
	identifyChannel *channels.IdentifyChannel
	onOffChannel    *channels.OnOffChannel
	device          *gatt.DiscoveredDevice

	service   gatt.ServiceDescription
	readChar  gatt.CharacteristicDescription
	writeChar gatt.CharacteristicDescription
	alertChar gatt.CharacteristicDescription
}

func NewBLETag(driver *BLETagDriver, device *gatt.DiscoveredDevice) error {

	address := device.Address

	if driver.FoundTags[address] {
		log.Infof("Already found tag %s", address)
		return nil
	}

	log.Infof("Found BLE Tag %s", address)
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

	if err := device.Connect(); err != nil {
		return fmt.Errorf("Connect Error: %s", err)
	}

	if err := device.DiscoverServices(); err != nil {
		return fmt.Errorf("Discovery Error: %s", err)
	}

	getChar := func(serviceUUID, charUUID string) (char gatt.CharacteristicDescription, err error) {

		service, ok := device.Services[serviceUUID]
		if !ok {
			return char, fmt.Errorf("Device %s does not have the StickNFind service %s", device.PublicAddress, serviceUUID)
		}

		chars, err := device.DiscoverCharacteristics(service)
		if err != nil {
			return char, fmt.Errorf("Discovery characteristics error: %s", err)
		}

		char, ok = chars[charUUID]
		if !ok {
			return char, fmt.Errorf("Device %s does not have the read characteristic %s", device.PublicAddress, charUUID)
		}

		return char, nil
	}

	if bt.readChar, err = getChar(stickNFindServiceUUID, stickNFindReadUUID); err != nil {
		return fmt.Errorf("Read characteristic not found: %s", err)
	}

	if bt.writeChar, err = getChar(stickNFindServiceUUID, stickNFindWriteUUID); err != nil {
		return fmt.Errorf("Write characteristic not found: %s", err)
	}

	if bt.alertChar, err = getChar("1802", "2a06"); err != nil {
		return fmt.Errorf("Alert characteristic not found: %s", err)
	}

	device.UpgradeSecurity()

	time.Sleep(time.Second * 3)

	log.Infof("Writing write char")
	device.Write(bt.writeChar, []byte{0x01, 0x02})

	time.Sleep(time.Second * 3)

	/*log.Infof("Writing alert char")
	device.Write(bt.alertChar, []byte{0x02})

	time.Sleep(time.Second * 3)

	log.Infof("Writing by handle char")
	device.WriteByHandle(0x001b, []byte{0x02})*/

	//device.Disconnect()

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
	return <-fp.device.Read(fp.readChar)
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

func (fp *BLETag) Buzz() (state chan bool, err chan error) {

	fp.Lock()

	state, err = make(chan bool, 2), make(chan error, 1)

	go func() {

		defer fp.Unlock()

		if !fp.driver.running {
			err <- fmt.Errorf("Driver not running, but received identify command")
			return
		}

		e := fp.device.Connect()
		if e != nil {
			err <- fmt.Errorf("Connect error:%s", e)
			return
		}

		log.Infof("SENDING BY HANDLE")
		fp.device.WriteByHandle(0x001b, []byte{0x02})
		state <- true
		time.Sleep(time.Second * 3)

		fp.device.WriteByHandle(0x001b, []byte{0x00})
		state <- false

		fp.device.Disconnect()

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
