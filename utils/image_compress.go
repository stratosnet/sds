package utils

import (
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"path/filepath"

	"github.com/nfnt/resize"
)

// ImageCommpress ImageCommpress
func ImageCommpress(pathStr string) (string, error) {
	fileIn, err := os.Open(pathStr)
	defer fileIn.Close()
	if CheckError(err) {
		ErrorLog(err)
		return "", err
	}
	lastPaths := filepath.Dir(pathStr)
	fileSuffix := path.Ext(pathStr)
	lastPaths = lastPaths + "/" + "tmp" + fileSuffix

	var origin image.Image

	if fileSuffix == ".png" {
		origin, err = png.Decode(fileIn)
		if CheckError(err) {
			origin, err = jpeg.Decode(fileIn)
		}
	} else {
		origin, err = jpeg.Decode(fileIn)
	}
	if CheckError(err) {
		ErrorLog(err)
		return "", err
	}

	fileOut, err := os.Create(lastPaths)
	if CheckError(err) {
		ErrorLog(err)
		return "", err
	}
	defer fileOut.Close()
	b := origin.Bounds()
	oWidth := b.Max.X
	DebugLog("oWidthoWidthoWidth", oWidth)
	oHeight := b.Max.Y
	DebugLog("oHeightoHeightoHeight", oHeight)
	width := uint(300)
	height := float32((300.0 / float32(oWidth))) * float32(oHeight)
	DebugLog("heightheightheight", height)
	canvas := resize.Thumbnail(width, uint(height), origin, resize.Bicubic)
	err = png.Encode(fileOut, canvas)
	if CheckError(err) {
		ErrorLog(err)
		return "", err
	}
	return lastPaths, nil
}
