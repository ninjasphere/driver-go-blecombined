// This package is a wrapper for the bluez gatt command
// which we are using as an interum measure to enable
// buzzing the sticknfind tags.
package bluez

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/juju/loggo"
)

var (
	log       = loggo.GetLogger("bluez")
	charRegex = regexp.MustCompile("handle = (?P<handle>[0-9a-fx]+), char properties = (?P<char_props>[0-9a-fx]+), char value handle = (?P<char_value_handle>[0-9a-fx]+), uuid = (?P<uuid>[0-9a-f-]+)")
	readRegex = regexp.MustCompile(`Characteristic value\/descriptor: ([0-9a-f ]+)`)
)

// gatttool -b F6:5F:20:4C:B0:DB -t random -l medium --char-write -a 0x002f -n 0103

const (
	bluezGattPath = "/usr/bin/gatttool"

	AddrTypePublic = "public"
	AddrTypeRandom = "random"
)

// GattCmd this is the handle to a bluez gatt cmd.
type GattCmd struct {
	baddr, addrType string
}

// Characteristic holds the attributes needed to address a characteristic via the api.
type Characteristic struct {
	UUID            string
	Handle          string // use this to READ
	CharValueHandle string // use this to WRITE
}

// ReadCharacteristics query a device for it's characteristics
func (gc *GattCmd) ReadCharacteristics() ([]*Characteristic, error) {

	chars := []*Characteristic{}

	data, err := run(bluezGattPath, "-b", gc.baddr, "-t", gc.addrType, "-l", "medium", "--characteristics")

	if err != nil {
		return chars, err
	}

	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		params := getAttributes(scanner.Text())

		c := &Characteristic{
			UUID:            params["uuid"],
			Handle:          params["handle"],
			CharValueHandle: params["char_value_handle"],
		}

		chars = append(chars, c)

		log.Infof(spew.Sprintf("%#v", c))
	}

	return chars, nil
}

// ReadCharacteristic connect to a ble device and read the caractersitic using the handle
func (gc *GattCmd) ReadCharacteristic(handle string) ([]byte, error) {

	payload := []byte{}

	data, err := run(bluezGattPath, "-b", gc.baddr, "-t", gc.addrType, "-l", "medium", "--char-read", "-a", handle)

	if err != nil {
		return payload, err
	}

	// extract the value
	value := decodeValue(data)

	// strip spaces
	value = strings.Replace(value, " ", "", -1)

	return hex.DecodeString(value)
}

// WriteCharacteristic connect to a ble device and write a value to the caractersitic using the handle
func (gc *GattCmd) WriteCharacteristic(handle, value string) error {

	_, err := run(bluezGattPath, "-b", gc.baddr, "-t", gc.addrType, "-l", "medium", "--char-write", "-a", handle, "-n", value)

	return err
}

// NewGattCmd create a gatt cmd handler.
func NewGattCmd(baddr, addrType string) (error *GattCmd) {

	// TODO is this a valid address / type

	return &GattCmd{baddr, addrType}
}

func run(cmd string, params ...string) (string, error) {

	cmdExec := exec.Command(cmd, params...)

	log.Infof("exec %s %v", cmd, params)

	var out bytes.Buffer
	cmdExec.Stdout = &out
	cmdExec.Stderr = &out
	err := cmdExec.Run()
	if err != nil {
		return out.String(), err
	}
	log.Debugf("run: %q\n", out.String())

	if strings.Contains(out.String(), "Connection refused") {
		return out.String(), fmt.Errorf("Connection refused.")
	}

	return out.String(), nil
}

func getAttributes(line string) map[string]string {

	matches := charRegex.FindAllStringSubmatch(line, -1)

	if matches == nil {
		return nil
	}

	params := make(map[string]string)

	for i, attr := range charRegex.SubexpNames() {
		if attr == "" {
			continue
		}
		params[attr] = matches[0][i]
	}

	return params
}

func decodeValue(line string) string {
	m := readRegex.FindStringSubmatch(line)

	if len(m) == 2 {
		return m[1]
	}

	return ""
}
