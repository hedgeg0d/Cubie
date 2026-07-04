package cube

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"cubie/connection"
)

type ScanResult = connection.ScanResult

func Scan(timeout time.Duration) ([]ScanResult, error) {
	return connection.Scan(timeout)
}

type CubeType int

const (
	WeilongV10AI CubeType = 1
)

var weilongSolvedState = [18]byte{0, 0, 0, 36, 146, 73, 73, 36, 146, 109, 182, 219, 146, 73, 36, 182, 219, 109}

const (
	moveHistorySize = 5
	burstWindow     = 300 * time.Millisecond
	burstThreshold  = 6
	burstIdle       = 400 * time.Millisecond
	resyncTick      = 40 * time.Millisecond
	stateTimeout    = 600 * time.Millisecond
)

var moveTable = map[int]string{
	0: "F", 1: "F'", 2: "B", 3: "B'",
	4: "U", 5: "U'", 6: "D", 7: "D'",
	8: "L", 9: "L'", 10: "R", 11: "R'",
}

type Quaternion struct {
	W, X, Y, Z float64
}

type Cube struct {
	Type     CubeType
	Power    int
	OnMove   func(move string)
	OnState  func(state [18]byte, solved bool)
	OnGyro   func(q Quaternion)
	OnResync func(state [18]byte)

	conn         *connection.Connection
	state        [18]byte
	stateChan    chan []byte
	powerChan    chan byte
	resyncStop   chan struct{}
	moveTimes    []time.Time
	lastMoveTime time.Time
	pendingSync  bool
	lastMoves    [5]string
	lastCounter  byte
	movePrimed   bool
	gyro         Quaternion
	mu           sync.Mutex
	stateMu      sync.Mutex
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
		c.handleGyro(decrypted)
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

func parseMoveHistory(decrypted []byte) [5]string {
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
	return moves
}

func (c *Cube) handleMoves(decrypted []byte) {
	moves := parseMoveHistory(decrypted)
	counter := decrypted[11]

	c.mu.Lock()
	c.lastMoves = moves
	count := 1
	if c.movePrimed {
		count = int(byte(counter - c.lastCounter))
	}
	c.lastCounter = counter
	c.movePrimed = true
	c.trackBurst(count)
	c.mu.Unlock()

	if count <= 0 {
		return
	}
	if count > moveHistorySize {
		count = moveHistorySize
	}

	for i := count - 1; i >= 0; i-- {
		move := moves[i]
		fmt.Println("Move made: " + move)
		if c.OnMove != nil {
			c.OnMove(move)
		}
	}
}

func (c *Cube) trackBurst(count int) {
	if count <= 0 {
		return
	}
	now := time.Now()
	c.lastMoveTime = now
	overflow := count > moveHistorySize
	n := min(count, burstThreshold)
	for range n {
		c.moveTimes = append(c.moveTimes, now)
	}
	cutoff := now.Add(-burstWindow)
	trimmed := c.moveTimes[:0]
	for _, t := range c.moveTimes {
		if t.After(cutoff) {
			trimmed = append(trimmed, t)
		}
	}
	c.moveTimes = trimmed

	if overflow || len(c.moveTimes) >= burstThreshold {
		c.pendingSync = true
	}
}

func (c *Cube) resyncWorker(stop chan struct{}) {
	ticker := time.NewTicker(resyncTick)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			settled := c.pendingSync && time.Since(c.lastMoveTime) >= burstIdle
			if settled {
				c.pendingSync = false
			}
			c.mu.Unlock()
			if settled {
				c.Resync()
			}
		}
	}
}

func (c *Cube) Resync() bool {
	c.mu.Lock()
	before := c.lastMoveTime
	c.mu.Unlock()

	if !c.UpdateState() {
		return false
	}
	if c.OnResync != nil {
		c.mu.Lock()
		state := c.state
		c.mu.Unlock()
		c.OnResync(state)
	}

	c.mu.Lock()
	if c.lastMoveTime.After(before) {
		c.pendingSync = true
	}
	c.mu.Unlock()
	return true
}

func (c *Cube) handleGyro(d []byte) {
	if len(d) < 17 {
		return
	}
	q := Quaternion{
		W: gyroComponent(d[1:5]),
		X: gyroComponent(d[5:9]),
		Z: -gyroComponent(d[9:13]),
		Y: gyroComponent(d[13:17]),
	}
	c.mu.Lock()
	c.gyro = q
	c.mu.Unlock()
	if c.OnGyro != nil {
		c.OnGyro(q)
	}
}

func gyroComponent(b []byte) float64 {
	return float64(int32(binary.LittleEndian.Uint32(b))) / float64(int64(1)<<30)
}

func (c *Cube) Gyro() Quaternion {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gyro
}

func (c *Cube) FindAndConnect(mac string) error {
	if c.Type != WeilongV10AI {
		return fmt.Errorf("unsupported cube type %d", c.Type)
	}
	c.mu.Lock()
	c.movePrimed = false
	c.moveTimes = nil
	c.pendingSync = false
	c.mu.Unlock()
	conn, err := connection.Setup(mac, int(c.Type), c.handleNotification)
	if err != nil {
		log.Println("Failed to connect to ", c.Type, " with address ", mac)
		return err
	}
	c.conn = conn
	stop := make(chan struct{})
	c.mu.Lock()
	c.resyncStop = stop
	c.mu.Unlock()
	go c.resyncWorker(stop)
	return nil
}

func (c *Cube) Disconnect() error {
	c.mu.Lock()
	if c.resyncStop != nil {
		close(c.resyncStop)
		c.resyncStop = nil
	}
	c.mu.Unlock()
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

func (c *Cube) UpdateState() bool {
	if c.conn == nil {
		return false
	}
	if c.Type != WeilongV10AI {
		return false
	}
	c.stateMu.Lock()
	drain(c.stateChan)
	stateReq := make([]byte, 20)
	stateReq[0] = 0xA3
	c.conn.SendData(stateReq)
	var received []byte
	select {
	case received = <-c.stateChan:
	case <-time.After(stateTimeout):
	}
	c.stateMu.Unlock()
	if received == nil {
		return false
	}
	c.mu.Lock()
	c.state = [18]byte(received)
	solved := c.state == weilongSolvedState
	c.mu.Unlock()
	if c.OnState != nil {
		c.OnState(c.state, solved)
	}
	return true
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

func (c *Cube) LastMovesList() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, 5)
	for i := 4; i >= 0; i-- {
		out = append(out, c.lastMoves[i])
	}
	return out
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
