package core

const (
	// This is either a client creating a connection, or a temporary connection made for a handshake
	// Read the first message from the connection. It should indicate what kind of connection it is
	ConnFirstMsgSize  = 14 // Conn type (8) + server port (2) + channel ID (4)
	ConnTypeClient    = "client__"
	ConnTypeHandshake = "handshke"

	HandshakeMessage = "sds_handshake"

	EncryptionHeaderSize = 12 // Nonce (8) + data length (4)
	EncryptionNonceSize  = 8
	EncryptionLengthSize = 4
)
