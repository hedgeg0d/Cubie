package controller

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

type Controller struct {
	uinputFile *os.File
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
	UI_SET_EVBIT   = 0x40045564
	UI_SET_KEYBIT  = 0x40045565
	UI_SET_ABSBIT  = 0x40045567
	UI_DEV_CREATE  = 0x5501
	UI_DEV_DESTROY = 0x5502

	ABS_X          = 0x00
	ABS_Y          = 0x01
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

	ioctl(c.uinputFile.Fd(), UI_SET_ABSBIT, ABS_X)
	ioctl(c.uinputFile.Fd(), UI_SET_ABSBIT, ABS_Y)

	setupAbs(c.uinputFile.Fd(), ABS_X, -32768, 32767)
	setupAbs(c.uinputFile.Fd(), ABS_Y, -32768, 32767)

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

	if _, _, err := unix.Syscall(unix.SYS_IOCTL, c.uinputFile.Fd(), UI_DEV_CREATE, 0); err != 0 {
		return err
	}

	return nil
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

func (c *Controller) emit(eventType, code, value int32) {
	ev := input_event{
		Time:  unix.Timeval{},
		Type:  uint16(eventType),
		Code:  uint16(code),
		Value: value,
	}
	buf := bytes.Buffer{}
	binary.Write(&buf, binary.LittleEndian, &ev)
	_, err := c.uinputFile.Write(buf.Bytes())
	if err != nil {
		log.Printf("Ошибка отправки события: %v", err)
	}
	if eventType != EV_SYN {
		c.sync()
	}
}

func (c *Controller) sync() {
	c.emit(EV_SYN, SYN_REPORT, 0)
}

func (c *Controller) PressA(holdTime time.Duration) {
	c.emit(EV_KEY, BTN_A, 1)
	time.Sleep(holdTime * time.Millisecond)
	c.emit(EV_KEY, BTN_A, 0)
}

func (c *Controller) PressB(holdTime time.Duration) {
	c.emit(EV_KEY, BTN_B, 1)
	time.Sleep(holdTime * time.Millisecond)
	c.emit(EV_KEY, BTN_B, 0)
}

func (c *Controller) PressX(holdTime time.Duration) {
	c.emit(EV_KEY, BTN_X, 1)
	time.Sleep(holdTime * time.Millisecond)
	c.emit(EV_KEY, BTN_X, 0)
}

func (c *Controller) PressY(holdTime time.Duration) {
	c.emit(EV_KEY, BTN_Y, 1)
	time.Sleep(holdTime * time.Millisecond)
	c.emit(EV_KEY, BTN_Y, 0)
}

func (c *Controller) HoldRight() {
	c.emit(EV_KEY, BTN_DPAD_RIGHT, 1)
}

func (c *Controller) ReleaseRight() {
	c.emit(EV_KEY, BTN_DPAD_RIGHT, 0)
}

func ioctl(fd uintptr, req uint, data uintptr) error {
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(req), data); err != 0 {
		return err
	}
	return nil
}
