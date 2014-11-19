package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/ninjasphere/go-ninja/logger"
	"strings"
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
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
		running:         false,
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
			if w.running == true {
				wplog.Debugf("Woohoo waypoint driver is running")
				time.Sleep(time.Second)
				for id, active := range w.activeWaypoints {
					wplog.Debugf("Waypoint %s is active? %t", id, active)
				}
				wplog.Debugf("%d waypoint(s) active", len(w.activeWaypoints))
				w.publishMessage("$location/waypoints", len(w.activeWaypoints))
			}
		}
	}()
}

func (w *WaypointDriver) handleSphereWaypoint(device *gatt.DiscoveredDevice) {
	if w.activeWaypoints[device.Address] {
		wplog.Debugf("waypoint %s already handled", device.Address)
		return
	}

	if device.Advertisement.LocalName != "NinjaSphereWaypoint" {
		return
	}

	if device.Connected == nil {
		device.Connected = func() {
			wplog.Debugf("Connected to waypoint: %s", device.Address)
			w.client.Notify(device.Address, true, waypointStartHandle, waypointEndHandle, true, false)
		}

		device.Disconnected = func() {
			wplog.Debugf("Disconnected from waypoint: %s", device.Address)
			w.activeWaypoints[device.Address] = false
		}

		device.Notification = func(notification *gatt.Notification) {
			wplog.Debugf("Got RSSI notification!")

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
	if err != nil {
		wplog.Errorf("Connect error:%s", err)
		return
	}

	w.activeWaypoints[device.Address] = true
}

func (w *WaypointDriver) publishMessage(topic string, packet interface{}) {
	p, err := json.Marshal(packet)
	if err == nil {
		w.conn.GetMqttClient().Publish(mqtt.QoS(0), topic, p)
	} else {
		wplog.Fatalf("marshalling error for %v", packet)
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
