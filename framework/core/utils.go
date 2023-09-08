package core

import (
	"encoding/binary"
	"io"
	"math/rand"
	"net"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/utils/encryption"
)

func WriteFull(c net.Conn, data []byte) error {
	n, err := c.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.Errorf("Couldn't write expected number of bytes: expected [%v] actual [%v]", len(data), n)
	}
	return nil
}

func CreateFirstMessage(connType string, ip net.IP, serverPort uint16, channelId uint32) []byte {
	buffer := make([]byte, ConnFirstMsgSize)
	copy(buffer[:8], []byte(connType)[:8])

	if ip.To16() != nil {
		copy(buffer[8:24], ip.To16())
	}

	binary.BigEndian.PutUint16(buffer[24:26], serverPort)
	binary.BigEndian.PutUint32(buffer[26:30], channelId)
	return buffer
}

func ParseFirstMessage(data []byte) (string, net.IP, uint16, uint32, error) {
	if len(data) != ConnFirstMsgSize {
		return "", nil, 0, 0, errors.Errorf("Invalid first message size [%v]", len(data))
	}
	connType := string(data[:8])
	ip := net.IP(data[8:24])
	serverPort := binary.BigEndian.Uint16(data[24:26])
	channelId := binary.BigEndian.Uint32(data[26:30])
	return connType, ip, serverPort, channelId, nil
}

func Pack(privKey, plaintext []byte) ([]byte, error) {
	// set nonce to 0 when message is non-encrypted packed
	packHead := make([]byte, EncryptionHeaderSize)
	if privKey != nil {
		_, err := rand.Read(packHead[:EncryptionNonceSize])
		if err != nil {
			return nil, err
		}
		nonce := binary.BigEndian.Uint64(packHead[:EncryptionNonceSize])
		ciphertext, err := encryption.EncryptAES(privKey, plaintext, nonce)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint32(packHead[EncryptionNonceSize:], uint32(len(ciphertext))) // Add encrypted data length
		return append(packHead, ciphertext...), nil
	} else {
		binary.BigEndian.PutUint64(packHead[:EncryptionNonceSize], uint64(0))
		binary.BigEndian.PutUint32(packHead[EncryptionNonceSize:], uint32(len(plaintext))) // Add encrypted data length
		return append(packHead, plaintext...), nil
	}

}

func unpackEncryptionHeader(data []byte) (uint64, uint32) {
	if len(data) < EncryptionHeaderSize {
		return 0, 0
	}

	nonce := binary.BigEndian.Uint64(data[:EncryptionNonceSize])
	length := binary.BigEndian.Uint32(data[EncryptionNonceSize:])

	return nonce, length
}

func ReadEncryptionHeader(c net.Conn) (nonce uint64, dataLen uint32, bytesRead int, err error) {
	buffer := make([]byte, EncryptionHeaderSize)
	if bytesRead, err = io.ReadFull(c, buffer); err != nil {
		return 0, 0, bytesRead, err
	}
	nonce, dataLen = unpackEncryptionHeader(buffer)
	return nonce, dataLen, bytesRead, nil
}

func Unpack(c net.Conn, privKey []byte, maxBodySize int) (plaintext []byte, bytesRead int, err error) {
	nonce, dataLen, bytesRead, err := ReadEncryptionHeader(c)
	if err != nil {
		return nil, bytesRead, err
	}
	if dataLen > uint32(maxBodySize) {
		return nil, bytesRead, errors.Errorf("message body is over sized [%v]", dataLen)
	}

	buffer := make([]byte, dataLen)
	n, err := io.ReadFull(c, buffer)
	bytesRead += n
	if err != nil {
		return nil, bytesRead, err
	}

	if privKey != nil {
		plaintext, err = encryption.DecryptAES(privKey, buffer, nonce, false)
		return plaintext, bytesRead, err
	}
	return buffer, bytesRead, err
}
