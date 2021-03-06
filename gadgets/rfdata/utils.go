package rfdata

import (
	"bytes"
	"encoding/hex"
	"strings"

	"github.com/jcw/flow"
)

func init() {
	flow.Registry["IntelHexToBin"] = func() flow.Circuitry { return &IntelHexToBin{} }
	flow.Registry["BinaryFill"] = func() flow.Circuitry { return &BinaryFill{} }
	flow.Registry["CalcCrc16"] = func() flow.Circuitry { return &CalcCrc16{} }
}

// IntelHexToBin takes lines of text and converts them to a large []byte value.
// Inserts an <addr> tag before the data. Registers as "IntelHexToBin".
type IntelHexToBin struct {
	flow.Gadget
	In  flow.Input
	Out flow.Output
}

// Start reading ":..." lines. Anything else causes the data to be flushed out.
func (w *IntelHexToBin) Run() {
	var buf bytes.Buffer
	for m := range w.In {
		if t, ok := m.(string); ok && strings.HasPrefix(t, ":") {
			b, err := hex.DecodeString(t[1:])
			flow.Check(err)
			// TODO: probably doesn't handle hex files over 64 KB
			if b[3] == 0 {
				if buf.Len() == 0 {
					addr := int(b[2]) + int(b[1])<<8
					w.Out.Send(flow.Tag{"<addr>", addr})
				}
				buf.Write(b[4 : 4+b[0]])
			}
		} else {
			if buf.Len() > 0 {
				w.Out.Send(buf.Bytes())
				buf.Reset()
			}
			w.Out.Send(m)
		}
	}
	if buf.Len() > 0 {
		w.Out.Send(buf.Bytes())
	}
}

// Take binary data and make sure it is filled to a specified multiple.
// Registers as "BinaryFill".
type BinaryFill struct {
	flow.Gadget
	In  flow.Input
	Len flow.Input
	Out flow.Output
}

// Start looking for []byte values, everything else is passed through unchanged.
func (w *BinaryFill) Run() {
	if n, ok := <-w.Len; ok {
		for m := range w.In {
			if data, ok := m.([]byte); ok {
				for n.(int) > 0 && len(data)%n.(int) != 0 {
					data = append(data, 0xFF)
				}
				m = data
			}
			w.Out.Send(m)
		}
	}
}

// CalcCrc16 takes []byte values and adds its CRC-16 as <crc16> tag after it.
// Registers as "CalcCrc16".
type CalcCrc16 struct {
	flow.Gadget
	In  flow.Input
	Out flow.Output
}

var crcTable = []uint16{
	0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401,
	0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400,
}

// Start looking for []byte values, everything else is passed through unchanged.
func (w *CalcCrc16) Run() {
	for m := range w.In {
		if data, ok := m.([]byte); ok {
			w.Out.Send(m)
			var crc uint16 = 0xFFFF
			for _, b := range data {
				crc = crc>>4 ^ crcTable[crc&0x0F] ^ crcTable[b&0x0F]
				crc = crc>>4 ^ crcTable[crc&0x0F] ^ crcTable[b>>4]
			}
			m = flow.Tag{"<crc16>", crc}
		}
		w.Out.Send(m)
	}
}
