//go:build !envtest

package gameproto

import (
	"bufio"
	"bytes"
	"testing"
)

func FuzzClassifyMinecraftHandshake(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x10, 0x00, 0xf2, 0x05, 0x09, 0x31, 0x32, 0x37, 0x2e, 0x30, 0x2e, 0x30, 0x2e, 0x31, 0x63, 0xdd, 0x01}) // valid login
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _, _ = classifyMinecraftHandshake(bufio.NewReader(bytes.NewReader(data)))
	})
}

func FuzzReadMinecraftVarInt(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x00})
	f.Add([]byte{0x01})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0x0f}) // max int32
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0x7f}) // invalid int32
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _ = readMinecraftVarInt(bytes.NewReader(data))
	})
}

func FuzzReadMinecraftString(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x04, 0x74, 0x65, 0x73, 0x74}) // "test"
	f.Add([]byte{0x00}) // empty string
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _ = readMinecraftString(bytes.NewReader(data))
	})
}

func FuzzClassifyTerrariaConnect(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x18, 0x00, 0x01, 0x0b, 0x54, 0x65, 0x72, 0x72, 0x61, 0x72, 0x69, 0x61, 0x32, 0x33, 0x30}) // valid connect
	f.Add([]byte{0x00})
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _, _ = classifyTerrariaConnect(bufio.NewReader(bytes.NewReader(data)))
	})
}

func FuzzParseTerrariaConnectRequest(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x0b, 0x54, 0x65, 0x72, 0x72, 0x61, 0x72, 0x69, 0x61, 0x32, 0x33, 0x30}) // valid string
	f.Add([]byte{0x00})
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _ = parseTerrariaConnectRequest(data)
	})
}

func FuzzReadTerrariaString(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x04, 0x74, 0x65, 0x73, 0x74}) // "test"
	f.Add([]byte{0x00})
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _ = readTerrariaString(bytes.NewReader(data))
	})
}

func FuzzReadTerraria7BitEncodedInt(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x00})
	f.Add([]byte{0x7f})
	f.Add([]byte{0x80, 0x01})
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(make([]byte, 65536))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", data, r)
			}
		}()
		_, _ = readTerrafia7BitEncodedInt(bytes.NewReader(data))
	})
}
