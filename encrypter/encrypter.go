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
		key, iv := getKeyAndIV(macAddress)
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
	return nil, errors.New("Unknown cube type")
}

func getKeyAndIV(macAddress string) ([]byte, []byte) {
	macClean := strings.ReplaceAll(macAddress, ":", "")
	macBytes, _ := hex.DecodeString(macClean)

	key := make([]byte, len(WEILONG_ROOT_KEY))
	copy(key, WEILONG_ROOT_KEY)
	iv := make([]byte, len(WEILONG_ROOT_IV))
	copy(iv, WEILONG_ROOT_IV)

	for i := range 6 {
		macIdx := 5 - i
		key[i] = byte((int(key[i]) + int(macBytes[macIdx])) % 255)
		iv[i] = byte((int(iv[i]) + int(macBytes[macIdx])) % 255)
	}
	return key, iv
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
