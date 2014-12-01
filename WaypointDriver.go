package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/ninjasphere/go-ninja/logger"

	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
	// "github.com/davecgh/go-spew/spew"
)

var wplog = logger.GetLogger("driver-go-waypoint")

type waypointPayload struct {
	Sequence    uint8
	AddressType uint8
	Rssi        int8
	Valid       uint8
}

type adPacket struct {
	Device   string `json:"device"`
	Waypoint string `json:"waypoint"`
	Rssi     int8   `json:"rssi"`
	IsSphere bool   `json:"isSphere"`
}

type ninjaPacket struct {
	Device   string `json:"device"`
	Waypoint string `json:"waypoint"`
	Rssi     int8   `json:"rssi"`
	IsSphere bool   `json:"isSphere"`
	name     string `json:"name,omitempty"`
}

type WaypointDriver struct {
	conn            *ninja.Connection
	sendEvent       func(event string, payload interface{}) error
	client          *gatt.Client
	activeWaypoints map[string]bool
	running         bool
}

func (w *WaypointDriver) sendRssi(device string, name string, waypoint string, rssi int8, isSphere bool) {
	device = strings.ToUpper(device)

	wplog.Debugf(">> Device:%s Waypoint:%s Rssi: %d", device, waypoint, rssi)

	ninjaPacket := ninjaPacket{
		Device:   device,
		Waypoint: waypoint,
		Rssi:     rssi,
		IsSphere: isSphere,
		name:     name,
	}

	w.conn.SendNotification("$device/"+device+"/TEMPPATH/rssi", ninjaPacket)

}

func NewWaypointDriver(client *gatt.Client) (*WaypointDriver, error) {

	conn, err := ninja.Connect("Waypoint")

	if err != nil {
		wplog.Fatalf("Failed to create Waypoint driver: %s", err)
		return nil, err
	}

	myWaypointDriver := &WaypointDriver{
		conn:            conn,
		client:          client,
		activeWaypoints: make(map[string]bool),
		running:         true,
	}

	err = conn.ExportDriver(myWaypointDriver)

	if err != nil {
		wplog.Fatalf("Failed to export waypoint driver: %s", err)
		return nil, err
	}

	myWaypointDriver.startWaypointLoop()

	return myWaypointDriver, nil
}

func (w *WaypointDriver) startWaypointLoop() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			if w.running == true {
				numWaypoints := 0
				for id, active := range w.activeWaypoints {
					log.Debugf("Waypoint %s is active? %t", id, active)
					if active {
						numWaypoints++
					}
				}
				w.conn.PublishRaw("$location/waypoints", numWaypoints)
			}
		}
	}()
}

func (w *WaypointDriver) handleSphereWaypoint(device *gatt.DiscoveredDevice) {
	if w.running {
		if w.activeWaypoints[device.Address] == true {
			wplog.Debugf("waypoint %s already handled", device.Address)
			return
		}

		if device.Advertisement.LocalName != "NinjaSphereWaypoint" {
			wplog.Debugf("device %s not actually sphere waypoint", device.Advertisement.LocalName)
			return
		}

		if device.Connected == nil {
			device.Connected = func() {
				wplog.Debugf("Connected to waypoint: %s", device.Address)
				w.client.Notify(device.Address, true, waypointStartHandle, waypointEndHandle, true, false)
				w.activeWaypoints[device.Address] = true
			}

			device.Disconnected = func() {
				wplog.Debugf("Disconnected from waypoint: %s", device.Address)
				w.activeWaypoints[device.Address] = false
			}

			device.Notification = func(notification *gatt.Notification) {

				var payload waypointPayload
				err := binary.Read(bytes.NewReader(notification.Data), binary.LittleEndian, &payload)
				if err != nil {
					wplog.Errorf("Failed to read waypoint payload : %s", err)
				}

				packet := &adPacket{
					Device:   fmt.Sprintf("%x", reverse(notification.Data[4:])),
					Waypoint: strings.Replace(device.Address, ":", "", -1),
					Rssi:     payload.Rssi,
					IsSphere: false,
				}

				w.sendRssi(packet.Device, "", packet.Waypoint, packet.Rssi, packet.IsSphere)
			}
		}

		err := w.client.Connect(device.Address, device.PublicAddress)
		wplog.Debugf("Connecting to sphere waypoint %s", device.PublicAddress)
		if err != nil {
			wplog.Errorf("Connect error:%s", err)
			return
		}

	}
}

func (d *WaypointDriver) GetModuleInfo() *model.Module {
	return ninja.LoadModuleInfo("./waypoint-package.json")
}

func (d *WaypointDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}

func (w *WaypointDriver) Start() error {
	wplog.Debugf("Starting waypoint driver")
	w.running = true
	return nil
}

func (w *WaypointDriver) Stop() error {
	w.running = false
	return nil
}

// reverse returns a reversed copy of u.
func reverse(u []byte) []byte {
	l := len(u)
	b := make([]byte, l)
	for i := 0; i < l/2+1; i++ {
		b[i], b[l-i-1] = u[l-i-1], u[i]
	}
	return b
}
