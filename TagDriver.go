package main

import (
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/driver-go-blecombined/bluez"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-ninja/suit"
)

var btinfo = ninja.LoadModuleInfo("./bletag-package.json")
var btlog = logger.GetLogger("driver-go-bletag")

type BLETagDriver struct {
	conn       *ninja.Connection
	sendEvent  func(event string, payload interface{}) error
	gattClient *gatt.Client
	running    bool
	FoundTags  map[string]bool
	lkConfig   sync.Mutex
	Config     *Config
}

func NewBLETagDriver(client *gatt.Client) (*BLETagDriver, error) {

	conn, err := ninja.Connect("BLETag")

	if err != nil {
		btlog.Fatalf("Failed to create BLE tag driver: %s", err)
		return nil, err
	}

	driver := &BLETagDriver{
		conn:       conn,
		gattClient: client,
		running:    true,
	}

	err = conn.ExportDriver(driver)

	if err != nil {
		btlog.Fatalf("Failed to export BLE tag driver: %s", err)
		return nil, err
	}

	driver.FoundTags = make(map[string]bool)

	return driver, nil
}

func (d *BLETagDriver) Configure(request *model.ConfigurationRequest) (*suit.ConfigurationScreen, error) {
	log.Infof("Incoming configuration request. Action:%s Data:%s", request.Action, string(request.Data))

	return nil, nil
}

func (d *BLETagDriver) GetModuleInfo() *model.Module {
	return btinfo
}

func (d *BLETagDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}

func (fp *BLETagDriver) Start(config *Config) error {
	btlog.Infof(spew.Sprintf("Starting BLE tag driver %v", config))
	fp.Config = config

	for _, tagConfig := range fp.Config.BleTags {
		NewBLETagFromConfig(fp, tagConfig)
	}

	fp.running = true
	return nil
}

func (fp *BLETagDriver) Stop() error {
	fp.running = false
	return nil
}

func (fp *BLETagDriver) saveNewTag(address string, publicAddress bool, readChar *bluez.Characteristic, alertChar *bluez.Characteristic) {

	fp.lkConfig.Lock()

	defer fp.lkConfig.Unlock()

	btlog.Debugf(spew.Sprintf("saveNewTag %s %t %#v %#v", address, publicAddress, readChar, alertChar))

	bleConfig := &BleTagConfig{
		Address:              address,
		PublicAddress:        publicAddress,
		ReadUUID:             readChar.UUID,
		ReadHandle:           readChar.Handle,
		ReadCharValueHandle:  readChar.CharValueHandle,
		AlertUUID:            alertChar.UUID,
		AlertHandle:          alertChar.Handle,
		AlertCharValueHandle: alertChar.CharValueHandle,
	}

	// replace in the list
	for i, bleTag := range fp.Config.BleTags {
		if bleTag.Address == bleConfig.Address {
			// delete that entry
			fp.Config.BleTags = append(fp.Config.BleTags[:i], fp.Config.BleTags[i+1:]...)
		}
	}

	// apend to the configuration
	fp.Config.BleTags = append(fp.Config.BleTags, bleConfig)

	btlog.Infof("saving configuration %#v", fp.Config)

	err := fp.sendEvent("config", fp.Config)

	if err != nil {
		btlog.Errorf("Error saving configuration: %s", err)
	}
}

// Config is persisted by HomeCloud, and provided when the app starts.
type Config struct {
	BleTags []*BleTagConfig `json:"bleTags"`
}

// BleTagConfig is persisted by HomeCloud, and provided when the app starts.
type BleTagConfig struct {
	Address       string `json:"address"`
	PublicAddress bool   `json:"publicAddress"`

	ReadUUID            string `json:"readUUID"`
	ReadHandle          string `json:"readHandle"`
	ReadCharValueHandle string `json:"readCharValueHandle"`

	AlertUUID            string `json:"alertUUID"`
	AlertHandle          string `json:"alertHandle"`
	AlertCharValueHandle string `json:"alertCharValueHandle"`
}
