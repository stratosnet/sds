package core

const (
	ConnFirstMsgSize  = 14 // Conn type (8) + server port (2) + channel ID (4)
	ConnTypeClient    = "client__"
	ConnTypeHandshake = "handshke"

	HandshakeMessage = "sds_handshake"

	EncryptionHeaderSize = 12 // Nonce (8) + data length (4)
	EncryptionNonceSize  = 8
	EncryptionLengthSize = 4
)
