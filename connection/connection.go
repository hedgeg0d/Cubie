package connection

import (
	"errors"
	"fmt"
	"log"
	"time"
	"cubie/encrypter"

	"tinygo.org/x/bluetooth"
)

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

var Conn Connection

func defaultCallback(buf []byte) {
	decrypted := Conn.crypt.Decrypt(buf)
	fmt.Print("Notification:")
	encrypter.PrintBytes(decrypted)
}

func customCallback(callback func (buf []byte)) func (buf []byte){
	newCallback := func (buf []byte) {
		callback(Conn.crypt.Decrypt(buf))
	}
	return newCallback
}

func get_callback(callback func(buf []byte)) func(buf []byte) {
	if callback == nil {
		return defaultCallback
	} else {
		return customCallback(callback)
	}
}

func Setup(macAddress string, cubeType int, callback func(buf []byte)) error {
	encrypter, err := encrypter.NewCubeEncrypter(macAddress, cubeType)
	if err != nil {
		log.Println(err)
		return err
	}
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Printf("Error turning on the bluetooth adapter: %s", err)
		return err
	}
	var device bluetooth.ScanResult
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if result.Address.String() == macAddress {
			log.Println("Cube found!" + result.Address.String())
			device = result
			adapter.StopScan()
		}
	})
	if err != nil {
		log.Printf("Error: Scan failed: %s", err)
		return err
	}

	conn, err := adapter.Connect(device.Address, bluetooth.ConnectionParams{})
	if err != nil {
		log.Printf("Connection error: %s", err)
		return err
	}
	log.Printf("Connected: %v\n", conn.Address)
	if cubeType == 1 {
		parsedUUID, err := bluetooth.ParseUUID(WEILONG_SERVICE_UUID)
		if err != nil {
			log.Println(err)
			return err
		}
		srvcs, err := conn.DiscoverServices([]bluetooth.UUID{parsedUUID})
		if err != nil {
			log.Printf("Error finding services: %s", err)
			return err
		}
		var chars []bluetooth.DeviceCharacteristic
		var writeUUID bluetooth.UUID
		if len(srvcs) > 0 {
			notifyUUID, err := bluetooth.ParseUUID(WEILONG_NOTIFY_UUID)
			if err != nil {
				log.Println(err)
				return err
			}
			writeUUID, err = bluetooth.ParseUUID(WEILONG_WRITE_UUID)
			if err != nil {
				log.Print(err)
				return err
			}

			chars, err = srvcs[0].DiscoverCharacteristics([]bluetooth.UUID{notifyUUID, writeUUID})
			if err != nil {
				log.Printf("Error discovering characteristics: %s", err)
				return err
			}
			log.Printf("Characteristics: %v\n", chars)

			for _, char := range chars {
				if char.UUID() == notifyUUID {
					callback = get_callback(callback)
					err := char.EnableNotifications(callback)
					if err != nil {
						log.Println("Error setting up notifications: ", err)
						return err
					}
					log.Println("Notifications have been set up")
				}
			}
		}
		Conn = Connection{
			device:    conn,
			crypt:     *encrypter,
			srvcs:     srvcs,
			chars:     chars,
			writeUUID: writeUUID,
		}
	} else {
		return errors.New("unknown cube type")
	}
	return nil
}

func SendData(data []byte) {
	for _, char := range Conn.chars {
		if char.UUID() == Conn.writeUUID {
			_, err := char.WriteWithoutResponse(Conn.crypt.Encrypt(data))
			if err != nil {
				log.Printf("Error sending data: %v\n", err)
				if data[0] == 0xA1 {
					log.Print("Retrying in 100 ms")
					time.Sleep(100 * time.Millisecond)
					_, err = char.WriteWithoutResponse(Conn.crypt.Encrypt(data))
					if err != nil {log.Println("Still failed: " + err.Error())} else {log.Println("Resolved!")}
				}
			}
		}
	}
}

func Disconnect() error {
    if len(Conn.srvcs) == 0 {
        return errors.New("not connected")
    }
    notifyUUID, err := bluetooth.ParseUUID(WEILONG_NOTIFY_UUID)
    if err != nil {
        log.Printf("Error parsing notify UUID: %v", err)
        return err
    }
    for _, char := range Conn.chars {
        if char.UUID() == notifyUUID {
            err := char.EnableNotifications(nil)
            if err != nil {
                log.Printf("Error disabling notification: %v", err)
            }
            break
        }
    }
    err = Conn.device.Disconnect()
    if err != nil {
        log.Printf("Error disconnecting: %v", err)
        return err
    }
    Conn = Connection{}
    return nil
}