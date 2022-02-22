package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stratosnet/sds/utils"
)

const HLS_SEGMENT_FILENAME = "%d.ts"

const HLS_HEADER_FILENAME = "index.m3u8"

const TEMP_FOLDER = "tmp"

var HlsInfoMap = make(map[string]*HlsInfo)

type HlsInfo struct {
	FileHash         string
	HeaderFile       string
	StartSliceNumber uint64
	TotalSize        int64
	SegmentToSlice   map[string]uint64
	SliceToSegment   map[uint64]string
}

func GetVideoDuration(path string) (uint64, error) {
	lengthCmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of",
		"default=noprint_wrappers=1:nokey=1", path)
	lengthOut, err := lengthCmd.Output()
	if err != nil {
		return 0, err
	}
	length, _ := strconv.ParseFloat(strings.TrimSuffix(string(lengthOut), "\n"), 64)
	return uint64(math.Ceil(length)), nil
}

func VideoToHls(fileHash string) bool {
	filePath := GetFilePath(fileHash)
	videoTmpFolder := GetVideoTmpFolder(fileHash)
	if _, err := os.Stat(videoTmpFolder); os.IsNotExist(err) {
		_ = os.Mkdir(videoTmpFolder, fs.ModePerm)
	}
	hlsSegmentFileName := videoTmpFolder + "/" + HLS_SEGMENT_FILENAME
	hlsHeaderFileName := videoTmpFolder + "/" + HLS_HEADER_FILENAME
	transformCmd := exec.Command("ffmpeg", "-i", filePath, "-codec:", "copy", "-start_number", "0", "-hls_time", "10",
		"-hls_list_size", "0", "-f", "hls", "-hls_segment_filename", hlsSegmentFileName, hlsHeaderFileName)
	stderr, _ := transformCmd.StderrPipe()
	transformCmd.Start()

	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		utils.Log(m)
	}
	transformCmd.Wait()
	return true
}

func GetHlsInfo(fileHash string, maxSliceCount uint64) (*HlsInfo, error) {
	videoTmpFolder := GetVideoTmpFolder(fileHash)
	totalSize := int64(0)

	files, err := ioutil.ReadDir(videoTmpFolder)
	if err != nil {
		utils.ErrorLog(err)
		return nil, err
	}

	sliceCount := len(files) - 1
	if sliceCount > int(maxSliceCount)-1 {
		utils.ErrorLog("Number of HLS slices exceeds number of arranged slices")
		return nil, err
	}

	startSliceNumber := maxSliceCount - uint64(sliceCount)
	currSliceNumber := startSliceNumber
	hlsInfo := &HlsInfo{
		FileHash:         fileHash,
		StartSliceNumber: startSliceNumber,
		SegmentToSlice:   make(map[string]uint64),
		SliceToSegment:   make(map[uint64]string),
	}

	for i := uint64(1); i < startSliceNumber; i++ {
		totalSize += int64(len(GetDumpySliceData(fileHash, i)))
	}

	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext == ".m3u8" {
			hlsInfo.HeaderFile = f.Name()
		}

		hlsInfo.SegmentToSlice[f.Name()] = currSliceNumber
		hlsInfo.SliceToSegment[currSliceNumber] = f.Name()
		currSliceNumber += 1
		totalSize += f.Size()
	}
	hlsInfo.TotalSize = totalSize
	return hlsInfo, nil
}

func LoadHlsInfo(fileHash, sliceHash, savePath string) *HlsInfo {
	slicePath := GetDownloadTmpPath(fileHash, sliceHash, savePath)
	data, err := ioutil.ReadFile(slicePath)
	if err != nil {
		utils.ErrorLog(err)
		return nil
	}
	var hlsInfo HlsInfo
	err = json.Unmarshal(data, &hlsInfo)
	if err != nil {
		utils.ErrorLog(err)
		return nil
	}
	return &hlsInfo
}

func GetVideoTmpFolder(fileHash string) string {
	return TEMP_FOLDER + "/" + fileHash
}

func GetDumpySliceData(fileHash string, sliceNumber uint64) []byte {
	return []byte(fmt.Sprintf("%v%d", fileHash, sliceNumber))
}
