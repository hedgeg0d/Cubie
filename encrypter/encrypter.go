package encrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type CubeEncrypter struct {
	key      []byte
	iv       []byte
	block    cipher.Block
	cubeType int
}

func NewCubeEncrypter(macAddress string, cubeType int) (*CubeEncrypter, error) {
	if cubeType == 1 {
		key, iv, err := getKeyAndIV(macAddress)
		if err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return &CubeEncrypter{
			key:      key,
			iv:       iv,
			block:    block,
			cubeType: cubeType,
		}, nil
	}
	return nil, errors.New("unknown cube type")
}

func getKeyAndIV(macAddress string) ([]byte, []byte, error) {
	macClean := strings.ReplaceAll(macAddress, ":", "")
	macBytes, err := hex.DecodeString(macClean)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid MAC address %q: %w", macAddress, err)
	}
	if len(macBytes) != 6 {
		return nil, nil, fmt.Errorf("invalid MAC address %q: expected 6 bytes, got %d", macAddress, len(macBytes))
	}

	key := make([]byte, len(WEILONG_ROOT_KEY))
	copy(key, WEILONG_ROOT_KEY)
	iv := make([]byte, len(WEILONG_ROOT_IV))
	copy(iv, WEILONG_ROOT_IV)

	for i := range 6 {
		macIdx := 5 - i
		key[i] = byte((int(key[i]) + int(macBytes[macIdx])) % 255)
		iv[i] = byte((int(iv[i]) + int(macBytes[macIdx])) % 255)
	}
	return key, iv, nil
}

func (c *CubeEncrypter) Encrypt(data []byte) []byte {
	if c.cubeType == 1 {
		result := make([]byte, len(data))
		copy(result, data)
		for i := range 16 {
			result[i] ^= c.iv[i]
		}
		c.block.Encrypt(result[:16], result[:16])
		if len(result) > 16 {
			offset := len(result) - 16
			for i := range 16 {
				result[offset+i] ^= c.iv[i]
			}
			c.block.Encrypt(result[offset:], result[offset:])
		}
		return result
	}
	return nil
}

func (c *CubeEncrypter) Decrypt(data []byte) []byte {
	if c.cubeType == 1 {
		result := make([]byte, len(data))
		copy(result, data)
		if len(result) > 16 {
			offset := len(result) - 16
			c.block.Decrypt(result[offset:], result[offset:])
			for i := range 16 {
				result[offset+i] ^= c.iv[i]
			}
		}
		c.block.Decrypt(result[:16], result[:16])
		for i := range 16 {
			result[i] ^= c.iv[i]
		}
		return result
	}
	return nil
}

func (c *CubeEncrypter) InfoRequest() []byte {
	bytes := make([]byte, 20)
	bytes[0] = 0xA1
	return c.Encrypt(bytes)
}

func PrintBytes(data []byte) {
	for i, b := range data {
		fmt.Printf("%02x", b)
		if i < len(data)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
}

var (
	WEILONG_ROOT_KEY = []byte{21, 119, 58, 92, 103, 14, 45, 31, 23, 103, 42, 19, 155, 103, 82, 87}
	WEILONG_ROOT_IV  = []byte{17, 35, 38, 37, 134, 42, 44, 59, 85, 6, 127, 49, 126, 103, 33, 87}
)
