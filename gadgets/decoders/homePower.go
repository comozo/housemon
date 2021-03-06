package decoders

import (
	"bytes"
	"encoding/binary"

	"github.com/jcw/flow"
)

func init() {
	flow.Registry["Node-homePower"] = func() flow.Circuitry { return &HomePower{} }
}

// Decoder for the "homePower.ino" sketch. Registers as "Node-homePower".
type HomePower struct {
	flow.Gadget
	In  flow.Input
	Out flow.Output
}

// Start decoding homePower packets.
func (w *HomePower) Run() {
	var vec, prev [6]uint16
	for m := range w.In {
		if v, ok := m.([]byte); ok && len(v) >= 12 {
			buf := bytes.NewBuffer(v[1:])
			err := binary.Read(buf, binary.LittleEndian, &vec)
			flow.Check(err)
			result := map[string]int{"<reading>": 1}
			if vec[0] != prev[0] {
				result["c1"] = int(vec[0])
				result["p1"] = time2watt(int(vec[1]))
			}
			if vec[2] != prev[2] {
				result["c2"] = int(vec[2])
				result["p2"] = time2watt(int(vec[3]))
			}
			if vec[4] != prev[4] {
				result["c3"] = int(vec[4])
				result["p3"] = time2watt(int(vec[5]))
			}
			copy(prev[:], vec[:])
			if len(result) == 1 {
				continue
			}
			m = result
		}

		w.Out.Send(m)
	}
}

func time2watt(t int) int {
	if t > 60000 {
		t = 1000 * (t - 60000)
	}
	if t > 0 {
		t = 18000000 / t
	}
	return t
}
