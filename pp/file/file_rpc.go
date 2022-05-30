package file

import (
	"io"
	"strings"
	"sync"
	"time"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/rpc"
)

const WAIT_TIMEOUT time.Duration = 3

var (
	reFileMutex sync.Mutex
	// key(fileHash) : value(fileSize)
	rpcFileInfoMap = make(map[string]uint64)
	// key(fileHash) : value(event)
	reFileEvent = make(map[string]rpc.Result)
	// key(fileHash) : value(pipe)
	pipes = make(map[string]pipe)
)

type pipe struct {
	reader   *io.PipeReader
	writer   *io.PipeWriter
}

// IsFileRpcRemote
func IsFileRpcRemote(hash string) bool {
	str := fileMap[hash]
	if str == "" {
		return false
	}
	return strings.Split(str, ":")[0] == "rpc"
}

// GetRemoteFileData
func GetRemoteFileData(hash string, offset *protos.SliceOffset) []byte {
	// compose event, as well notify the remote user
	r := &rpc.Result {
		Return: rpc.UPLOAD_DATA,
		OffsetStart: &offset.SliceOffsetStart,
		OffsetEnd: &offset.SliceOffsetEnd,
	}

	// send event and open the pipe for coming data
	reFileMutex.Lock()
	reFileEvent[hash] = *r
	var p pipe
	p.reader, p.writer = io.Pipe()
	pipes[hash] = p
	reFileMutex.Unlock()

	// read on the pipe
	data := make([]byte, offset.SliceOffsetEnd - offset.SliceOffsetStart)
	var cursor []byte
	var read uint64
	var done = make(chan bool)

	cursor = data[:]

	go func() {
		for {
			n, err := p.reader.Read(cursor)
			if err != nil {
				done <- false
				return
			}
			read = read + uint64(n)
			cursor = data[read:]
			if read >= offset.SliceOffsetEnd - offset.SliceOffsetStart {
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT * time.Second):
		return nil
	case s := <-done:
		if s {
			return []byte(data)
		}else {
			return nil
		}
	}
}

// GetRemoteFileSize
func GetRemoteFileSize(hash string) uint64{
	return rpcFileInfoMap[hash]
}

// SendFileDataBack the rpc handler writes data to slice upload task
func SendFileDataBack(hash string, content []byte) {
	reFileMutex.Lock()

	if w, found := pipes[hash]; found && w.writer != nil {
		pipes[hash].writer.Write(content)
	}

	reFileMutex.Unlock()
}

// SetRemoteFileResult a result is given to the remote client
func SetRemoteFileResult(hash string, result rpc.Result) {
	reFileMutex.Lock()
	reFileEvent[hash] = result
	reFileMutex.Unlock()
}

// SaveRemoteFileHash
func SaveRemoteFileHash(hash, fileName string, fileSize uint64) {
	fileMap[hash] = "rpc:" + fileName

	reFileMutex.Lock()
	rpcFileInfoMap[hash] = fileSize
	reFileMutex.Unlock()
}

// GetRemoteFileEvent
func GetRemoteFileEvent(hash string) (rpc.Result, bool) {
	reFileMutex.Lock()
	defer reFileMutex.Unlock()

	var result rpc.Result
	var found bool
	if result, found = reFileEvent[hash]; found {
		delete(reFileEvent, hash);
	}

	return result, found
}
