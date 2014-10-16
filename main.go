package main

import (
	"fmt"
	"github.com/ninjasphere/gatt"
	// "github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
)

var log = logger.GetLogger("driver-go-blecombined")
var fpDriver *FlowerPowerDriver
var wpDriver *WaypointDriver

func main() {

	log.Infof("BLE Driver Starting")
	out, err := exec.Command("hciconfig").Output()
	if err != nil {
		log.Errorf(fmt.Sprintf("Error: %s", err))
	}
	re := regexp.MustCompile("([0-9A-F]{2}\\:{0,1}){6}")
	mac := strings.Replace(re.FindString(string(out)), ":", "", -1)
	log.Infof("The local mac is %s\n", mac)

	client := &gatt.Client{
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

	client.Advertisement = handleAdvertisement

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
		wpDriver.handleSphereWaypoint(device)
		return
	}

	for uuid := range device.Advertisement.ServiceUuids {
		if uuid == flowerPowerServiceUuid {
			if fpDriver.announcedFlowerPowers[device.Address] {
				return
			}

			log.Infof("Making flower power %s", device.Address)
			err := NewFlowerPower(fpDriver, device)
			if err != nil {
				log.Errorf("Error creating FlowerPower device ", err)
			}
		}
	}
}
