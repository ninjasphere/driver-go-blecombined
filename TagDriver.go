package main

import (
	// "github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
)

var btinfo = ninja.LoadModuleInfo("./bletag-package.json")
var btlog = logger.GetLogger("driver-go-bletag")

type BLETagDriver struct {
	conn       *ninja.Connection
	sendEvent  func(event string, payload interface{}) error
	gattClient *gatt.Client
	running    bool
	FoundTags  map[string]bool
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

func (d *BLETagDriver) GetModuleInfo() *model.Module {
	return btinfo
}

func (d *BLETagDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}

func (fp *BLETagDriver) Start() error {
	btlog.Infof("Starting BLE tag driver")
	fp.running = true
	return nil
}

func (fp *BLETagDriver) Stop() error {
	fp.running = false
	return nil
}
