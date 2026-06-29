package cube

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"cubie/connection"
)

type CubeType int

const (
	WeilongV10AI CubeType = 1
)

type Cube struct {
	Type      CubeType
	Power     int
	state     [18]byte
	connceted bool
}

var stateChan = make(chan []byte)
var powerChan = make(chan byte)
var last_moves [5]string
var weilongSolvedState = [18]byte{0, 0, 0, 36, 146, 73, 73, 36, 146, 109, 182, 219, 146, 73, 36, 182, 219, 109}

var RCallback func() = func() {}
var LCallback func() = func() {}
var UCallback func() = func() {}
var DCallback func() = func() {}
var FCallback func() = func() {}
var BCallback func() = func() {}

var RrCallback func() = func() {}
var LrCallback func() = func() {}
var UrCallback func() = func() {}
var DrCallback func() = func() {}
var FrCallback func() = func() {}
var BrCallback func() = func() {}

func GetCallback(move string) *func() {
        switch move {
        case "R":	return &RCallback
        case "L":	return &LCallback
        case "U":	return &UCallback
        case "D":	return &DCallback
        case "F":	return &FCallback
        case "B":	return &BCallback
        case "R'":	return &RrCallback
        case "L'":	return &LrCallback
        case "U'":	return &UrCallback
        case "D'":	return &DrCallback
        case "F'":	return &FrCallback
        case "B'":	return &BrCallback
        default:	return nil
        }
}

// it recieves already decrypted data
func weilongCallback(decrypted []byte) {
	if decrypted[0] == 0xAB {
		return
	} else if decrypted[0] == 0xA5 {
		moveTable := map[int]string{
			0:  "F",
			1:  "F'",
			2:  "B",
			3:  "B'",
			4:  "U",
			5:  "U'",
			6:  "D",
			7:  "D'",
			8:  "L",
			9:  "L'",
			10: "R",
			11: "R'",
		}

		bitString := ""
		for _, b := range decrypted[12:16] {
			bitString += fmt.Sprintf("%08b", b)
		}

		moves := []string{}
		for i := range 5 {
			start := i * 5
			moveBits := bitString[start : start+5]
			moveCode, _ := strconv.ParseInt(moveBits, 2, 64)

			move, ok := moveTable[int(moveCode)]
			if ok {
				moves = append(moves, move)
			} else {
				moves = append(moves, fmt.Sprintf("Unknown(%d)", moveCode))
			}
			if i == 4 {
				last_moves = [5]string(moves)
				last_move := moves[0]
				fmt.Println("Move made: " + last_move)
				switch(last_move) {
					case "R": RCallback()
					case "L": LCallback()
					case "U": UCallback()
					case "D": DCallback()
					case "F": FCallback()
					case "B": BCallback()
					
					case "R'": RrCallback()
					case "L'": LrCallback()
					case "U'": UrCallback()
					case "D'": DrCallback()
					case "F'": FrCallback()
					case "B'": BrCallback()
				}
			}
		}
		return
	} else if decrypted[0] == 0xA3 {
		stateChan <- decrypted[1:19]
	} else if decrypted[0] == 0xA1 {
		modelName := string(bytes.Trim(decrypted[1:9], "\x00"))
		hwVersion := fmt.Sprintf("%d.%d", decrypted[9], decrypted[10])
		swVersion := fmt.Sprintf("%d.%d", decrypted[11], decrypted[12])

		fmt.Printf("Model: %s\n", modelName)
		fmt.Printf("Hardware version: %s\n", hwVersion)
		fmt.Printf("Software version: %s\n", swVersion)
		return
	} else if decrypted[0] == 0xA4 {
		fmt.Printf("Battery level: %d\n", decrypted[1])
		powerChan <- decrypted[1]
		return
	}
}

func (c *Cube) FindAndConnect(mac string) error {
	var callback func(buf []byte)
	if c.Type == WeilongV10AI {
		callback = weilongCallback
	} else {
		callback = nil
	}
	err := connection.Setup(mac, int(c.Type), callback)
	if err != nil {
		log.Println("Failed to connect to ", c.Type, " with address ", mac)
		return err
	}
	return nil
}

// For Weilong v10 AI it is required to send info request, to start communication with cube
// For QiYi Smart Cube it is required to send special packet
// Weilong v10 AI protocol documentation: https://github.com/lukeburong/weilong-v10-ai-protocol
// QiYi Smart Cube protocol documentation: https://github.com/Flying-Toast/qiyi_smartcube_protocol/
func (c *Cube) GreetCube() {
	if c.Type == WeilongV10AI {
		info_req := make([]byte, 20)
		info_req[0] = 0xA1
		connection.SendData(info_req)
	}
}

// QiYi Smart Cube sends this after greeting automatically
func (c *Cube) UpdatePowerInfo() {
	if c.Type == WeilongV10AI {
		power_req := make([]byte, 20)
		power_req[0] = 0xA4
		connection.SendData(power_req)
		perc := <-powerChan
		c.Power = int(perc)
	}
}

// QiYi Smart Cube doesn't have this packet. It sends cube state on each move,
// so handle it in notification callback function.
func (c *Cube) UpdateState() {
	if c.Type == WeilongV10AI {
		state_req := make([]byte, 20)
		state_req[0] = 0xA3
		connection.SendData(state_req)
		recieved := <-stateChan
		c.state = [18]byte(recieved)
	}
}

func (c *Cube) IsSolved() bool {
	if c.Type == WeilongV10AI && c.state == weilongSolvedState {
		return true
	}
	return false
}

func GetLastMoves() string {
	var res strings.Builder
	for i := 4; i >= 0; i-- {
		res.WriteString(last_moves[i] + " ")
	}
	return res.String()
}
