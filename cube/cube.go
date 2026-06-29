package cube

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"cubie/connection"
)

type CubeType int

const (
	WeilongV10AI CubeType = 1
)

var weilongSolvedState = [18]byte{0, 0, 0, 36, 146, 73, 73, 36, 146, 109, 182, 219, 146, 73, 36, 182, 219, 109}

var moveTable = map[int]string{
	0: "F", 1: "F'", 2: "B", 3: "B'",
	4: "U", 5: "U'", 6: "D", 7: "D'",
	8: "L", 9: "L'", 10: "R", 11: "R'",
}

type Cube struct {
	Type    CubeType
	Power   int
	OnMove  func(move string)
	OnState func(state [18]byte, solved bool)

	conn      *connection.Connection
	state     [18]byte
	stateChan chan []byte
	powerChan chan byte
	lastMoves [5]string
	mu        sync.Mutex
}

func New(t CubeType) *Cube {
	return &Cube{
		Type:      t,
		stateChan: make(chan []byte, 1),
		powerChan: make(chan byte, 1),
	}
}

func (c *Cube) handleNotification(decrypted []byte) {
	switch decrypted[0] {
	case 0xAB:
		return
	case 0xA5:
		c.handleMoves(decrypted)
	case 0xA3:
		trySend(c.stateChan, decrypted[1:19])
	case 0xA1:
		modelName := string(bytes.Trim(decrypted[1:9], "\x00"))
		hwVersion := fmt.Sprintf("%d.%d", decrypted[9], decrypted[10])
		swVersion := fmt.Sprintf("%d.%d", decrypted[11], decrypted[12])
		fmt.Printf("Model: %s\n", modelName)
		fmt.Printf("Hardware version: %s\n", hwVersion)
		fmt.Printf("Software version: %s\n", swVersion)
	case 0xA4:
		fmt.Printf("Battery level: %d\n", decrypted[1])
		trySend(c.powerChan, decrypted[1])
	}
}

func (c *Cube) handleMoves(decrypted []byte) {
	bitString := ""
	for _, b := range decrypted[12:16] {
		bitString += fmt.Sprintf("%08b", b)
	}

	moves := [5]string{}
	for i := range 5 {
		start := i * 5
		moveBits := bitString[start : start+5]
		moveCode, _ := strconv.ParseInt(moveBits, 2, 64)
		move, ok := moveTable[int(moveCode)]
		if ok {
			moves[i] = move
		} else {
			moves[i] = fmt.Sprintf("Unknown(%d)", moveCode)
		}
	}

	c.mu.Lock()
	c.lastMoves = moves
	c.mu.Unlock()

	lastMove := moves[0]
	fmt.Println("Move made: " + lastMove)
	if c.OnMove != nil {
		c.OnMove(lastMove)
	}
}

func (c *Cube) FindAndConnect(mac string) error {
	if c.Type != WeilongV10AI {
		return fmt.Errorf("unsupported cube type %d", c.Type)
	}
	conn, err := connection.Setup(mac, int(c.Type), c.handleNotification)
	if err != nil {
		log.Println("Failed to connect to ", c.Type, " with address ", mac)
		return err
	}
	c.conn = conn
	return nil
}

func (c *Cube) Disconnect() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Disconnect()
	c.conn = nil
	return err
}

func (c *Cube) GreetCube() {
	if c.Type == WeilongV10AI {
		infoReq := make([]byte, 20)
		infoReq[0] = 0xA1
		c.conn.SendData(infoReq)
	}
}

func (c *Cube) UpdatePowerInfo() {
	if c.Type == WeilongV10AI {
		drain(c.powerChan)
		powerReq := make([]byte, 20)
		powerReq[0] = 0xA4
		c.conn.SendData(powerReq)
		perc := <-c.powerChan
		c.Power = int(perc)
	}
}

func (c *Cube) UpdateState() {
	if c.Type == WeilongV10AI {
		drain(c.stateChan)
		stateReq := make([]byte, 20)
		stateReq[0] = 0xA3
		c.conn.SendData(stateReq)
		received := <-c.stateChan
		c.mu.Lock()
		c.state = [18]byte(received)
		solved := c.state == weilongSolvedState
		c.mu.Unlock()
		if c.OnState != nil {
			c.OnState(c.state, solved)
		}
	}
}

func (c *Cube) IsSolved() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Type == WeilongV10AI && c.state == weilongSolvedState
}

func (c *Cube) LastMoves() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var res strings.Builder
	for i := 4; i >= 0; i-- {
		res.WriteString(c.lastMoves[i] + " ")
	}
	return res.String()
}

func trySend[T any](ch chan T, v T) {
	select {
	case ch <- v:
	default:
	}
}

func drain[T any](ch chan T) {
	select {
	case <-ch:
	default:
	}
}
