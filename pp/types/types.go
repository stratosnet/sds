package types

var (
	P2P_SERVER_KEY             = ContextKey{Key: "PPServerKey"}
	LISTEN_OFFLINE_QUIT_CH_KEY = ContextKey{Key: "ListenOfflineQuitCh"}
	PP_NETWORK_KEY             = ContextKey{Key: "PpNetworkKey"}
)

type ContextKey struct {
	Key string
}
