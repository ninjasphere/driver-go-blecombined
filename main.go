package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"

	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/support"
)

var log = logger.GetLogger("driver-go-blecombined")
var fpDriver *FlowerPowerDriver
var wpDriver *WaypointDriver
var tagDriver *BLETagDriver
var client *gatt.Client //kill me
var sent = false

func main() {

	log.Infof("BLE Driver Starting")

	// reset BLE layer
	out, err := exec.Command("/opt/ninjablocks/bin/sphere-ble-reset", "status").Output()
	if err != nil {
		log.Errorf(fmt.Sprintf("error: while checking status of BLE connection: %s. ignoring reset logic.", err))
	} else {
		log.Errorf("error: detected case where BLE stack could not be started properly. signalling need for reset...")
		out, err = exec.Command("/opt/ninjablocks/bin/sphere-ble-reset", "signal-reset").Output()
		if err == nil {
			log.Errorf("signalling successful. blocking until stopped by reset logic.")
			support.WaitUntilSignal()
		} else {
			log.Errorf("signalling unsuccessful. continuing without waiting.")
		}
	}

	// use hciconfig to the get the mac address
	out, err = exec.Command("hciconfig").Output()
	if err != nil {
		log.Errorf(fmt.Sprintf("Error: %s", err))
	}
	re := regexp.MustCompile("([0-9A-F]{2}\\:{0,1}){6}")
	mac := strings.Replace(re.FindString(string(out)), ":", "", -1)
	log.Infof("The local mac is %s\n", mac)

	client = &gatt.Client{
		StateChange: func(newState string) {
			log.Infof("Client state change: %s", newState)
		},
	}

	fpDriver, err = NewFlowerPowerDriver(client)
	if err != nil {
		log.Errorf("Failed to create FlowerPower driver: ", err)
	}

	wpDriver, err = NewWaypointDriver(client)
	if err != nil {
		log.FatalError(err, "Failed to create waypoint driver")
	}

	tagDriver, err = NewBLETagDriver(client)
	if err != nil {
		log.FatalError(err, "Failed to create BLE Tag driver")
	}

	client.Advertisement = handleAdvertisement

	client.Rssi = func(address string, name string, rssi int8) {
		//log.Printf("Rssi update address:%s rssi:%d", address, rssi)
		wpDriver.sendRssi(strings.Replace(address, ":", "", -1), name, mac, rssi, true)
		//spew.Dump(device);
	}

	log.Infof("Starting client scan")
	err = client.Start()
	if err != nil {
		log.FatalError(err, "Failed to start client")
	}

	err = client.StartScanning(true)
	if err != nil {
		log.FatalError(err, "Failed to start scanning")
	}

	//----------------------------------------------------------------------------------------

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)
}

func handleAdvertisement(device *gatt.DiscoveredDevice) {

	if device.Advertisement.LocalName == "NinjaSphereWaypoint" {
		log.Infof("Found waypoint %s", device.Address)
		wpDriver.handleSphereWaypoint(device)
	}

	for uuid := range device.Advertisement.ServiceUuids {
		if uuid == flowerPowerServiceUuid {
			if fpDriver.announcedFlowerPowers[device.Address] {
				return
			}
			log.Infof("Found Flower Power %s", device.Address)
			err := NewFlowerPower(fpDriver, device)
			if err != nil {
				log.Errorf("Error creating FlowerPower device ", err)
			}
		}
	}

	// look for tags which are CLOSE to the sphere!!
	for uuid := range device.Advertisement.ServiceUuids {
		if uuid == stickNFindServiceUuid {
			if device.Rssi > minRSSI {
				err := NewBLETag(tagDriver, device)
				if err != nil {
					log.Errorf("Error creating BLE Tag device ", err)
				}
			}
		}
	}
}
