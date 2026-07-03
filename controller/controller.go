//go:build linux

package controller

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

type Controller struct {
	uinputFile *os.File
	mu         sync.Mutex
}

type input_event struct {
	Time  unix.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type uinput_user_dev struct {
	Name      [80]byte
	Id        input_id
	FFEffects uint32
	AbsMax    [64]int32
	AbsMin    [64]int32
	AbsFuzz   [64]int32
	AbsFlat   [64]int32
}

type input_id struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

const (
	UI_SET_EVBIT  = 0x40045564
	UI_SET_KEYBIT = 0x40045565
	UI_SET_ABSBIT = 0x40045567
	UI_DEV_CREATE = 0x5501
	UI_DEV_DESTROY = 0x5502

	ABS_X          = 0x00
	ABS_Y          = 0x01
	ABS_Z          = 0x02
	ABS_RX         = 0x03
	ABS_RY         = 0x04
	ABS_RZ         = 0x05
	BTN_A          = 0x130
	BTN_B          = 0x131
	BTN_X          = 0x133
	BTN_Y          = 0x134
	BTN_TL         = 0x136
	BTN_TR         = 0x137
	BTN_SELECT     = 0x13a
	BTN_START      = 0x13b
	BTN_DPAD_UP    = 0x13c
	BTN_DPAD_DOWN  = 0x13d
	BTN_DPAD_LEFT  = 0x13e
	BTN_DPAD_RIGHT = 0x13f
	EV_KEY         = 0x01
	EV_ABS         = 0x03
	EV_SYN         = 0x00
	SYN_REPORT     = 0x00

	UI_ABS_SETUP = 0x40145570
)

var Actions = []string{
	"A", "B", "X", "Y",
	"LB", "RB", "Select", "Start",
	"DPad Up", "DPad Down", "DPad Left", "DPad Right",
}

var actionCodes = map[string]uint16{
	"A":          BTN_A,
	"B":          BTN_B,
	"X":          BTN_X,
	"Y":          BTN_Y,
	"LB":         BTN_TL,
	"RB":         BTN_TR,
	"Select":     BTN_SELECT,
	"Start":      BTN_START,
	"DPad Up":    BTN_DPAD_UP,
	"DPad Down":  BTN_DPAD_DOWN,
	"DPad Left":  BTN_DPAD_LEFT,
	"DPad Right": BTN_DPAD_RIGHT,
}

var Axes = []string{"Left X", "Left Y", "Right X", "Right Y", "LT", "RT"}

var axisCodes = map[string]uint16{
	"Left X":  ABS_X,
	"Left Y":  ABS_Y,
	"Right X": ABS_RX,
	"Right Y": ABS_RY,
	"LT":      ABS_Z,
	"RT":      ABS_RZ,
}

func AxisCode(name string) (uint16, bool) {
	code, ok := axisCodes[name]
	return code, ok
}

func (c *Controller) Init() error {
	uinputFile, err := os.OpenFile("/dev/uinput", os.O_WRONLY|unix.O_NONBLOCK, 0666)
	if err != nil {
		return err
	}
	c.uinputFile = uinputFile

	ioctl(c.uinputFile.Fd(), UI_SET_EVBIT, EV_KEY)
	ioctl(c.uinputFile.Fd(), UI_SET_EVBIT, EV_ABS)
	ioctl(c.uinputFile.Fd(), UI_SET_EVBIT, EV_SYN)

	buttons := []uint16{BTN_A, BTN_B, BTN_X, BTN_Y, BTN_TL, BTN_TR, BTN_SELECT, BTN_START, BTN_DPAD_UP, BTN_DPAD_DOWN, BTN_DPAD_LEFT, BTN_DPAD_RIGHT}
	for _, btn := range buttons {
		ioctl(c.uinputFile.Fd(), UI_SET_KEYBIT, uintptr(btn))
	}

	axes := []uint32{ABS_X, ABS_Y, ABS_Z, ABS_RX, ABS_RY, ABS_RZ}
	for _, axis := range axes {
		ioctl(c.uinputFile.Fd(), UI_SET_ABSBIT, uintptr(axis))
		setupAbs(c.uinputFile.Fd(), axis, -32768, 32767)
	}

	var uidev uinput_user_dev
	copy(uidev.Name[:], "Xbox Virtual Controller\x00")
	uidev.Id = input_id{
		Bustype: unix.BUS_USB,
		Vendor:  0x045e,
		Product: 0x02d1,
		Version: 0x01,
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, &uidev); err != nil {
		return err
	}
	if _, err := c.uinputFile.Write(buf.Bytes()); err != nil {
		return err
	}

	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, c.uinputFile.Fd(), UI_DEV_CREATE, 0); errno != 0 {
		return errno
	}

	return nil
}

func (c *Controller) Close() error {
	if c.uinputFile == nil {
		return nil
	}
	unix.Syscall(unix.SYS_IOCTL, c.uinputFile.Fd(), UI_DEV_DESTROY, 0)
	err := c.uinputFile.Close()
	c.uinputFile = nil
	return err
}

func setupAbs(fd uintptr, code uint32, min, max int32) {
	absSetup := struct {
		Code uint32
		Val  int32
		Min  int32
		Max  int32
		Fuzz int32
		Flat int32
	}{
		Code: code,
		Min:  min,
		Max:  max,
	}
	ioctl(fd, UI_ABS_SETUP, uintptr(unsafe.Pointer(&absSetup)))
}

func (c *Controller) writeEvent(eventType, code, value int32) {
	ev := input_event{
		Type:  uint16(eventType),
		Code:  uint16(code),
		Value: value,
	}
	buf := bytes.Buffer{}
	binary.Write(&buf, binary.LittleEndian, &ev)
	if _, err := c.uinputFile.Write(buf.Bytes()); err != nil {
		log.Printf("Error emitting event: %v", err)
	}
}

func (c *Controller) emitKey(code, value int32) {
	if c.uinputFile == nil {
		return
	}
	c.mu.Lock()
	c.writeEvent(EV_KEY, code, value)
	c.writeEvent(EV_SYN, SYN_REPORT, 0)
	c.mu.Unlock()
}

func (c *Controller) Press(action string, holdTime time.Duration) {
	code, ok := actionCodes[action]
	if !ok || c.uinputFile == nil {
		return
	}
	c.emitKey(int32(code), 1)
	time.Sleep(holdTime * time.Millisecond)
	c.emitKey(int32(code), 0)
}

func (c *Controller) SetButton(action string, down bool) {
	code, ok := actionCodes[action]
	if !ok {
		return
	}
	value := int32(0)
	if down {
		value = 1
	}
	c.emitKey(int32(code), value)
}

func (c *Controller) SetAxis(code uint16, value int32) {
	if c.uinputFile == nil {
		return
	}
	if value < -32768 {
		value = -32768
	}
	if value > 32767 {
		value = 32767
	}
	c.mu.Lock()
	c.writeEvent(EV_ABS, int32(code), value)
	c.writeEvent(EV_SYN, SYN_REPORT, 0)
	c.mu.Unlock()
}

func (c *Controller) Frame(f func(w func(eventType, code, value int32))) {
	if c.uinputFile == nil {
		return
	}
	c.mu.Lock()
	f(c.writeEvent)
	c.writeEvent(EV_SYN, SYN_REPORT, 0)
	c.mu.Unlock()
}

func ioctl(fd uintptr, req uint, data uintptr) error {
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(req), data); err != 0 {
		return err
	}
	return nil
}
