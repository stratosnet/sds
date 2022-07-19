package core

const (
	ConnFirstMsgSize  = 14 // Conn type (8) + server port (2) + channel ID (4)
	ConnTypeClient    = "client__"
	ConnTypeHandshake = "handshke"

	HandshakeMessage = "sds_handshake"

	EncryptedHeaderSize = 12 // Nonce (8) + data length (4)
	EncryptedNonceSize  = 8
	EncryptedLengthSize = 4
)
