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

func CreateFirstMessage(connType string, serverPort uint16, channelId uint32) []byte {
	buffer := make([]byte, ConnFirstMsgSize)
	copy(buffer[:8], []byte(connType)[:8])
	binary.BigEndian.PutUint16(buffer[8:10], serverPort)
	binary.BigEndian.PutUint32(buffer[10:14], channelId)
	return buffer
}

func ParseFirstMessage(data []byte) (string, uint16, uint32, error) {
	if len(data) != ConnFirstMsgSize {
		return "", 0, 0, errors.Errorf("Invalid first message size [%v]", len(data))
	}
	connType := string(data[:8])
	serverPort := binary.BigEndian.Uint16(data[8:10])
	channelId := binary.BigEndian.Uint32(data[10:14])
	return connType, serverPort, channelId, nil
}

func EncryptAndPack(privKey, plaintext []byte) ([]byte, error) {
	header := make([]byte, EncryptionHeaderSize) // Contains nonce and encrypted data length
	if _, err := rand.Read(header[:EncryptionNonceSize]); err != nil {
		return nil, err
	}
	nonce := binary.BigEndian.Uint64(header[:EncryptionNonceSize])

	ciphertext, err := encryption.EncryptAES(privKey, plaintext, nonce)
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint32(header[EncryptionNonceSize:], uint32(len(ciphertext))) // Add encrypted data length

	return append(header, ciphertext...), nil
}

func Pack(plaintext []byte) ([]byte, error) {
	// set nonce to 0 when message is non-encrypted packed
	header := make([]byte, EncryptionHeaderSize)
	binary.BigEndian.PutUint64(header[:EncryptionNonceSize], uint64(0))
	binary.BigEndian.PutUint32(header[EncryptionNonceSize:], uint32(len(plaintext))) // Add encrypted data length
	return append(header, plaintext...), nil
}

func UnpackEncryptionHeader(data []byte) (uint64, uint32) {
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
	nonce, dataLen = UnpackEncryptionHeader(buffer)
	return nonce, dataLen, bytesRead, nil
}

func ReadEncryptedHeaderAndBody(c net.Conn, privKey []byte, maxBodySize int) (plaintext []byte, bytesRead int, err error) {
	nonce, dataLen, bytesRead, err := ReadEncryptionHeader(c)
	if err != nil {
		return nil, bytesRead, err
	}
	if dataLen > uint32(maxBodySize) {
		return nil, bytesRead, errors.Errorf("encrypted message body is over sized [%v]", dataLen)
	}

	buffer := make([]byte, dataLen)
	n, err := io.ReadFull(c, buffer)
	bytesRead += n
	if err != nil {
		return nil, bytesRead, err
	}

	plaintext, err = encryption.DecryptAES(privKey, buffer, nonce)
	return plaintext, bytesRead, err
}

func ReadNonEncryptedHeaderAndBody(c net.Conn, maxBodySize int) (plaintext []byte, bytesRead int, err error) {
	_, dataLen, bytesRead, err := ReadEncryptionHeader(c)
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

	return buffer, bytesRead, err
}
