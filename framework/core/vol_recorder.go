package core

const (
	InboundLog  VolRecType = 1
	OutboundLog VolRecType = 2
	ReadLog     VolRecType = 3
	WriteLog    VolRecType = 4
)

type VolRecType int

// Equal compares two VolRecType instances
func (t VolRecType) Equal(t2 VolRecType) bool {
	return t == t2
}

// WriteCloser
type VolRecorder interface {
	Record(VolRecType)
}

func (sc *ServerConn) Record(volRecType VolRecType) {
	switch volRecType {
	case InboundLog:

		break
	case OutboundLog:

		break
	case ReadLog:

		break
	case WriteLog:

		break
	}

}
