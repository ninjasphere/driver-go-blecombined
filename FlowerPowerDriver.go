package main

import (
	// "github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
)

var info = ninja.LoadModuleInfo("./flowerpower-package.json")
var fplog = logger.GetLogger("driver-go-flowerpower")

type FlowerPowerDriver struct {
	conn                  *ninja.Connection
	sendEvent             func(event string, payload interface{}) error
	gattClient            *gatt.Client
	running               bool
	announcedFlowerPowers map[string]bool
}

func NewFlowerPowerDriver(client *gatt.Client) (*FlowerPowerDriver, error) {
	announcedFlowerPowers := make(map[string]bool)

	conn, err := ninja.Connect("FlowerPower")

	if err != nil {
		fplog.Fatalf("Failed to create Flower Power driver: %s", err)
		return nil, err
	}

	driver := &FlowerPowerDriver{
		conn:                  conn,
		gattClient:            client,
		running:               false,
		announcedFlowerPowers: announcedFlowerPowers,
	}

	err = conn.ExportDriver(driver)

	if err != nil {
		fplog.Fatalf("Failed to export FlowerPower driver: %s", err)
		return nil, err
	}

	return driver, nil
}

func (d *FlowerPowerDriver) GetModuleInfo() *model.Module {
	return info
}

func (d *FlowerPowerDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}

func (fp *FlowerPowerDriver) Start() error {
	fplog.Infof("Starting FlowerPower driver")
	fp.running = true
	return nil
}

func (fp *FlowerPowerDriver) Stop() error {
	fp.running = false
	return nil
}
