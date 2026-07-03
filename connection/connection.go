package connection

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"cubie/encrypter"

	"tinygo.org/x/bluetooth"
)

type ScanResult struct {
	Address string
	Name    string
	RSSI    int16
}

func Scan(timeout time.Duration) ([]ScanResult, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Printf("Error turning on the bluetooth adapter: %s", err)
		return nil, err
	}

	var mu sync.Mutex
	found := map[string]ScanResult{}

	go func() {
		time.Sleep(timeout)
		adapter.StopScan()
	}()

	err := adapter.Scan(func(a *bluetooth.Adapter, r bluetooth.ScanResult) {
		name := r.LocalName()
		if name == "" {
			return
		}
		mu.Lock()
		found[r.Address.String()] = ScanResult{Address: r.Address.String(), Name: name, RSSI: r.RSSI}
		mu.Unlock()
	})
	if err != nil {
		log.Printf("Error: Scan failed: %s", err)
		return nil, err
	}

	mu.Lock()
	out := make([]ScanResult, 0, len(found))
	for _, r := range found {
		out = append(out, r)
	}
	mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].RSSI > out[j].RSSI })
	return out, nil
}

const (
	WEILONG_SERVICE_UUID = "0783b03e-7735-b5a0-1760-a305d2795cb0"
	WEILONG_NOTIFY_UUID  = "0783b03e-7735-b5a0-1760-a305d2795cb1"
	WEILONG_WRITE_UUID   = "0783b03e-7735-b5a0-1760-a305d2795cb2"
)

type Connection struct {
	device    bluetooth.Device
	crypt     encrypter.CubeEncrypter
	srvcs     []bluetooth.DeviceService
	chars     []bluetooth.DeviceCharacteristic
	writeUUID bluetooth.UUID
}

func wrapCallback(crypt *encrypter.CubeEncrypter, callback func(buf []byte)) func(buf []byte) {
	if callback == nil {
		return func(buf []byte) {
			decrypted := crypt.Decrypt(buf)
			fmt.Print("Notification:")
			encrypter.PrintBytes(decrypted)
		}
	}
	return func(buf []byte) {
		callback(crypt.Decrypt(buf))
	}
}

func Setup(macAddress string, cubeType int, callback func(buf []byte)) (*Connection, error) {
	crypt, err := encrypter.NewCubeEncrypter(macAddress, cubeType)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if cubeType != 1 {
		return nil, errors.New("unknown cube type")
	}

	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Printf("Error turning on the bluetooth adapter: %s", err)
		return nil, err
	}
	var device bluetooth.ScanResult
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if result.Address.String() == macAddress {
			log.Println("Cube found! " + result.Address.String())
			device = result
			adapter.StopScan()
		}
	})
	if err != nil {
		log.Printf("Error: Scan failed: %s", err)
		return nil, err
	}

	conn, err := adapter.Connect(device.Address, bluetooth.ConnectionParams{})
	if err != nil {
		log.Printf("Connection error: %s", err)
		return nil, err
	}
	log.Printf("Connected: %v\n", conn.Address)

	parsedUUID, err := bluetooth.ParseUUID(WEILONG_SERVICE_UUID)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	srvcs, err := conn.DiscoverServices([]bluetooth.UUID{parsedUUID})
	if err != nil {
		log.Printf("Error finding services: %s", err)
		return nil, err
	}
	if len(srvcs) == 0 {
		return nil, errors.New("cube service not found")
	}

	notifyUUID, err := bluetooth.ParseUUID(WEILONG_NOTIFY_UUID)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	writeUUID, err := bluetooth.ParseUUID(WEILONG_WRITE_UUID)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	chars, err := srvcs[0].DiscoverCharacteristics([]bluetooth.UUID{notifyUUID, writeUUID})
	if err != nil {
		log.Printf("Error discovering characteristics: %s", err)
		return nil, err
	}
	log.Printf("Characteristics: %v\n", chars)

	wrapped := wrapCallback(crypt, callback)
	for _, char := range chars {
		if char.UUID() == notifyUUID {
			if err := char.EnableNotifications(wrapped); err != nil {
				log.Println("Error setting up notifications: ", err)
				return nil, err
			}
			log.Println("Notifications have been set up")
		}
	}

	return &Connection{
		device:    conn,
		crypt:     *crypt,
		srvcs:     srvcs,
		chars:     chars,
		writeUUID: writeUUID,
	}, nil
}

func (c *Connection) SendData(data []byte) {
	for _, char := range c.chars {
		if char.UUID() == c.writeUUID {
			_, err := char.WriteWithoutResponse(c.crypt.Encrypt(data))
			if err != nil {
				log.Printf("Error sending data: %v\n", err)
				if data[0] == 0xA1 {
					log.Print("Retrying in 100 ms")
					time.Sleep(100 * time.Millisecond)
					_, err = char.WriteWithoutResponse(c.crypt.Encrypt(data))
					if err != nil {
						log.Println("Still failed: " + err.Error())
					} else {
						log.Println("Resolved!")
					}
				}
			}
		}
	}
}

func (c *Connection) Disconnect() error {
	if len(c.srvcs) == 0 {
		return nil
	}
	notifyUUID, err := bluetooth.ParseUUID(WEILONG_NOTIFY_UUID)
	if err != nil {
		log.Printf("Error parsing notify UUID: %v", err)
		return err
	}
	for _, char := range c.chars {
		if char.UUID() == notifyUUID {
			if err := char.EnableNotifications(nil); err != nil {
				log.Printf("Error disabling notification: %v", err)
			}
			break
		}
	}
	err = c.device.Disconnect()
	*c = Connection{}
	if err != nil {
		log.Printf("Error disconnecting: %v", err)
		return err
	}
	return nil
}
