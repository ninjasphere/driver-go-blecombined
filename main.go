package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gatt"
	// "github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"time"
)

var log = logger.GetLogger("driver-go-blecombined")
var fpDriver *FlowerPowerDriver
var wpDriver *WaypointDriver
var client *gatt.Client //kill me
var sent = false

func main() {

	log.Infof("BLE Driver Starting")
	out, err := exec.Command("hciconfig").Output()
	if err != nil {
		log.Errorf(fmt.Sprintf("Error: %s", err))
	}
	re := regexp.MustCompile("([0-9A-F]{2}\\:{0,1}){6}")
	mac := strings.Replace(re.FindString(string(out)), ":", "", -1)
	log.Infof("The local mac is %s\n", mac)

	// client := &gatt.Client{
	// 	StateChange: func(newState string) {
	// 		log.Infof("Client state change: %s", newState)
	// 	},
	// }

	client = &gatt.Client{
		StateChange: func(newState string) {
			log.Infof("Client state change: %s", newState)
		},
	}

	fpDriver, err = NewFlowerPowerDriver(client)
	if err != nil {
		log.Errorf("Failed to create FlowerPower driver: ", err)
	}
	//
	// wpDriver, err = NewWaypointDriver(client)
	// if err != nil {
	// 	log.FatalError(err, "Failed to create waypoint driver")
	// }

	client.Advertisement = handleAdvertisement

	log.Debugf("Starting client scan")
	err = client.Start()
	if err != nil {
		log.FatalError(err, "Failed to start client")
	}

	err = client.StartScanning(true)
	if err != nil {
		log.FatalError(err, "Failed to start scanning")
	}

	// testdevice := client.CreateDeviceByAddress("C0:10:5E:A6:50:7F")
	// testdevice.Connected = func() {
	// 	log.Debugf("Connected to tag: %s", testdevice.Address)
	// 	if !sent {
	// 		log.Debugf("Beeping tag")
	// 		cmds := make([]string, 1)
	// 		cmds[0] = "121b0002"
	// 		client.SendRawCommands(testdevice.Address, cmds)
	// 		time.Sleep(time.Second * 5)
	// 		cmds[0] = "121b0000"
	// 		client.SendRawCommands(testdevice.Address, cmds)
	// 		sent = true
	// 		time.Sleep(time.Second * 5)
	// 	}
	// }
	//
	// testdevice.Disconnected = func() {
	// 	log.Debugf("Disconnected from tag: %s", testdevice.Address)
	// }
	//
	// err = client.Connect(testdevice.Address, testdevice.PublicAddress)
	// if err != nil {
	// 	log.Errorf("Connect error:%s", err)
	// 	return
	// }

	//----------------------------------------------------------------------------------------

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)
}

func handleAdvertisement(device *gatt.DiscoveredDevice) {
	// if device.Advertisement.LocalName == "NinjaSphereWaypoint" {
	// 	log.Debugf("Found waypoint %s", device.Address)
	// 	wpDriver.handleSphereWaypoint(device)
	// 	return
	// }
	//
	// for uuid := range device.Advertisement.ServiceUuids {
	// 	if uuid == flowerPowerServiceUuid {
	// 		if fpDriver.announcedFlowerPowers[device.Address] {
	// 			return
	// 		}
	// 		log.Debugf("Found Flower Power %s", device.Address)
	// 		err := NewFlowerPower(fpDriver, device)
	// 		if err != nil {
	// 			log.Errorf("Error creating FlowerPower device ", err)
	// 		}
	// 	}
	// }

	for uuid := range device.Advertisement.ServiceUuids {
		if uuid == stickNFindServiceUuid {
			log.Debugf("Found sticknfind at address " + device.Address)
			if !sent {
				handleTag(device)
			}
		}
	}

}

func handleTag(device *gatt.DiscoveredDevice) {

	if device.Connected == nil {
		device.Connected = func() {
			log.Debugf("Connected to tag: %s", device.Address)
			if !sent {
				log.Debugf("Beeping tag")
				cmds := make([]string, 1)
				cmds[0] = "121b0002"
				client.SendRawCommands(device.Address, cmds)
				time.Sleep(time.Second * 5)
				cmds[0] = "121b0000"
				client.SendRawCommands(device.Address, cmds)
				sent = true
				time.Sleep(time.Second * 5)
			}
		}

		device.Disconnected = func() {
			log.Debugf("Disconnected from tag: %s", device.Address)
		}

		log.Debugf("Connecting to tag...")
		spew.Dump(device)
		err := client.Connect(device.Address, device.PublicAddress)
		if err != nil {
			log.Errorf("Connect error:%s", err)
			return
		}

	}
}
