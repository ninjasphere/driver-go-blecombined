package bluez

import (
	"encoding/hex"
	"testing"
)

const (
	sampleReadResult = "022d00d0d09e38a534ddb8c04f04685213a78d"
)

func TestCharacteristicsRegex(t *testing.T) {
	line := "handle = 0x0002, char properties = 0x0a, char value handle = 0x0003, uuid = 00002a00-0000-1000-8000-00805f9b34fb"

	params := getAttributes(line)

	if params["handle"] != "0x0002" {
		t.Errorf("bad handle %v", params["handle"])
	}

	if params["char_value_handle"] != "0x0003" {
		t.Errorf("bad char_value_handle %v", params["char_value_handle"])
	}

	if params["uuid"] != "00002a00-0000-1000-8000-00805f9b34fb" {
		t.Errorf("bad characteristic uuid %v", params["uuid"])
	}
}

func TestHexDecode(t *testing.T) {

	payload, err := hex.DecodeString(sampleReadResult)

	if err != nil {
		t.Error(err)
	}

	if len(payload) != 19 {
		t.Errorf("bad payload %x", payload)
	}

}
