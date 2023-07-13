// Based on the firmata platform.
// Package client provies a client for interacting with microcontrollers
// using the sparki-learning firmware, and serves as a replacement for it's
// provided python controller application:
//   https://sparki-learning.readthedocs.io/en/latest
//   https://github.com/radarjd/sparki_learning

package client

import (
	"errors"
	"io"
	"log"
	"sync/atomic"
	"strconv"
	"time"
	"gobot.io/x/gobot/v2"
)

// myro Codes
const (
	SYNC		byte = 0x16
	ETB		byte = 0x17
	BadCommand	byte = 0x5B
	LCDClear	byte = 0x30
	LCDDrawPixel	byte = 0x33
	LCDDrawString	byte = 0x35
	LCDPrint	byte = 0x36
	LCDPrintLn	byte = 0x37
	LCDReadPixel	byte = 0x38
	LCDUpdate	byte = 0x39
	Motors		byte = 0x41
	BackwardCM	byte = 0x42
	ForwardCM	byte = 0x43
	Ping		byte = 0x44
	ReceiveIR	byte = 0x45
	SendIR		byte = 0x46
	Servo		byte = 0x47
	SetDebugLevel	byte = 0x48
	SetRGBLED	byte = 0x49
	SetStatusLED	byte = 0x4A
	Stop		byte = 0x4B
	TurnBy		byte = 0x4C
	GetName		byte = 0x4F
	SetName		byte = 0x50
	ReadEEPROM	byte = 0x51
	WriteEEPROM	byte = 0x52
	LCDSetColor	byte = 0x54
	NOOP		byte = 0x5A
	Beep		byte = 0x62
	Compass		byte = 0x63
	Gamepad		byte = 0x65
	GetAccel	byte = 0x65
	GetLight	byte = 0x6B
	GetLine		byte = 0x6D
	GetMag		byte = 0x6F
	GripperClose	byte = 0x76
	GripperOpen	byte = 0x78
	GripperStop	byte = 0x79
	Init		byte = 0x7A
)

// Errors
var (
	ErrConnected = errors.New("client is already connected")
)

// Client represents a client connection to a firmata board
type Client struct {
	FirmwareName    string
	ProtocolVersion string
	connected       atomic.Value
	connection      io.ReadWriteCloser
	ConnectTimeout  time.Duration
	gobot.Eventer
}

// New returns a new Client
func New() *Client {
	c := &Client{
		ProtocolVersion: "",
		FirmwareName:    "",
		connection:      nil,
		ConnectTimeout:  5 * time.Second,
		Eventer:         gobot.NewEventer(),
	}

	c.connected.Store(false)

	return c
}

func (b *Client) setConnected(c bool) {
	b.connected.Store(c)
}

func (b *Client) haltFunctions() {
	b.Stop()
	b.SetRGBLED(0, 0, 0)
	b.SetStatusLED(0)
	b.LCDClear(true)
}

// Disconnect disconnects the Client
func (b *Client) Disconnect() error {
	b.haltFunctions()
	b.setConnected(false)
	return b.connection.Close()
}

func (b *Client) clearSync() error {
	_,err := b.read(1)
	return err
}

// Connected returns the current connection state of the Client
func (b *Client) Connected() bool {
	return b.connected.Load().(bool)
}

// Connect connects to the Client given conn. It first resets the firmata board
// then continuously polls the firmata board for new information when it's
// available.
func (b *Client) Connect(conn io.ReadWriteCloser) error {
	if b.Connected() {
		return ErrConnected
	}

	b.connection = conn
	err := b.Reset()
	if err != nil {
		return err
	}
	connected := make(chan bool, 1)
	connectError := make(chan error, 1)

	// start it off...
	log.Println("[CLIENT] Connect Starting...")
	b.sendInit()
	if err != nil {
		return err
	}

	go func() {
		for {
			e := b.receive()
			if e != nil {
				connectError <- e
				return
			}
			b.setConnected(true)
			connected <- true
			break
		}
	}()

	select {
	case <-connected:
	case e := <-connectError:
		return e
	case <-time.After(b.ConnectTimeout):
		return errors.New("unable to connect. Perhaps you need to flash your Arduino with Firmata?")
	}

	time.Sleep(10 * time.Millisecond)
	//Uncomment for initial signs of working connection...
	//b.SetRGBLED(90, 100, 0)
	//b.MoveForward(5.1)
	//b.sendNOOP()

	log.Println("[CLIENT] Connected!")
	return nil
}

func (b *Client) Reset() error {
	return nil
}

func (b *Client) sendBad() error {
	err := b.transmitSync([]byte{BadCommand})
	return err
}

func (b *Client) sendInit() error {
	return b.transmit([]byte{Init})
}

func (b *Client) sendNOOP() error {
	return b.transmitSync([]byte{NOOP})
}

func (b *Client) DrawPixel(x uint, y uint) error {
	err := b.transmit([]byte{LCDDrawPixel})
	if err == nil {
		err = b.transmit(uintCharArray(x))
	}
	if err == nil {
		err = b.transmitSync(uintCharArray(y))
	}
	return err
}

func (b *Client) EnableGamepad() error {
	return b.transmitSync([]byte{Gamepad})
}

func (b *Client) LCDClear(update bool) error {
	err := b.transmitSync([]byte{LCDClear})
	if err == nil && update {
		err = b.LCDUpdate()
	}
	return err
}

func (b *Client) LCDUpdate() error {
	return b.transmitSync([]byte{LCDUpdate})
}

func (b *Client) SetRGBLED(red uint, green uint, blue uint) error {
	err := b.transmit([]byte{SetRGBLED})
	if err == nil {
		err = b.transmit(uintCharArray(red))
	}
	if err == nil {
		err = b.transmit(uintCharArray(green))
	}
	if err == nil {
		err = b.transmitSync(uintCharArray(blue))
	}
	return err
}

// set the status LED to @brightness,
// @brightness should be between 0 and 100 (as a percentage)
func (b *Client) SetStatusLED(brightness uint) error {
	err := b.transmit([]byte{SetStatusLED})
	if err == nil {
		err = b.transmitSync(uintCharArray(brightness))
	}
	return err
}

// DigitalWrite writes value to pin.
// Hack to show led.Toggle() working...
func (b *Client) DigitalWrite(pin int, value int) error {
	if value > 0 {
		return b.SetStatusLED(100)
	} else {
		return b.SetStatusLED(0)
	}
}

func (b *Client) MoveBackward(cm float32) error {
	err := b.transmit([]byte{BackwardCM})
	if err == nil {
		err = b.transmitSync(floatCharArray(cm))
	}
	return err
}

func (b *Client) MoveForward(cm float32) error {
	err := b.transmit([]byte{ForwardCM})
	if err == nil {
		err = b.transmitSync(floatCharArray(cm))
	}
	return err
}

// moves Sparki's left wheel at @left and right wheel at @right speed for
// @time; speed should be a number from 1 to 100 indicating the percentage
// of power used, time should be in seconds; if time < 0, move immediately
// and without stopping
func (b *Client) Move(left int, right int, secs float32) error {
	err := b.transmit([]byte{Motors})
	if err == nil {
		err = b.transmit(intCharArray(left))
	}
	if err == nil {
		err = b.transmit(intCharArray(right))
	}
	if err == nil {
		err = b.transmitSync(floatCharArray(secs))
	}
	return err
}

func (b *Client) Stop() error {
	return b.transmitSync([]byte{Stop})
}

func (b *Client) notransmit(data []byte) error {
	data = append(data, ETB)
	log.Println("[CLIENT TX]", data)
	return nil
}

func (b *Client) transmit(data []byte) error {
	// MAX_TRANSMISSION = 20
	data = append(data, ETB)
	log.Println("[CLIENT TX]", data)
	_, err := b.connection.Write(data[:])
	return err
}

func (b *Client) transmitSync(data []byte) error {
	err := b.transmit(data)
	if err == nil {
		b.clearSync()
	}
	return err
}

func (b *Client) read(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(b.connection, buf)
	return buf, err
}

func (b *Client) receive() error {
	inCount := 0
	var inBuffer []byte

	for {
		inByte, err := b.read(1)
		if err != nil {
			return err
		}
		if inByte[0] == ETB {
			break
		}
		inCount++
		inBuffer = append(inBuffer, inByte[0])
	}
	log.Printf("[CLIENT RX] bytes %d: %s", inCount, inBuffer)
	log.Println("[CLIENT RX] data ", inBuffer)
	b.clearSync()

	return nil
}

func floatCharArray(value float32) []byte {
	return []byte(strconv.FormatFloat(float64(value), 'f', -1, 32))
}

func intCharArray(value int) []byte {
	return []byte(strconv.Itoa(value))
}

func uintCharArray(value uint) []byte {
	return intCharArray(int(value))
}
