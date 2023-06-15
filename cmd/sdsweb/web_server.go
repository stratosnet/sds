package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

//go:embed build
var EmbeddedAssets embed.FS
var UnpackedAssets fs.FS

// fsFunc is shorthand for constructing a http.FileSystem
// implementation
type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// AssetHandler returns an http.Handler that will serve files from
// the UnpackedAssets.  When locating a file, it will strip the given
// prefix from the request and prepend the root to the filesystem
// lookup: typical prefix might be /web/, and root would be build.
func AssetHandler(prefix, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		// If we can't find the asset, return the default index.html
		// content
		f, err := UnpackedAssets.Open(assetPath)
		if os.IsNotExist(err) {
			return UnpackedAssets.Open(filepath.Join(root, "index.html"))
		}

		// Otherwise assume this is a legitimate request routed
		// correctly
		return f, err
	})
	return http.StripPrefix(prefix, http.FileServer(http.FS(handler)))
}

func startWebServer() error {
	if err := unpackWebDirectory(); err != nil {
		return err
	}

	if setting.Config.WebServer.Port == "" {
		return errors.New("Missing configuration for web server port")
	}
	if _, err := strconv.ParseUint(setting.Config.WebServer.Port, 10, 64); err != nil {
		return errors.Wrap(err, "failed to parse UI port configuration")
	}

	if err := UpdateWebConfig(); err != nil {
		return err
	}

	UnpackedAssets = os.DirFS(setting.Config.WebServer.Path)
	handler := AssetHandler("", "")

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/*filepath", handler)

	go func() {
		utils.Log("Starting UI Server...")
		err := http.ListenAndServe(fmt.Sprintf(":%v", setting.Config.WebServer.Port), mux)
		if err != nil {
			utils.ErrorLog("Error in UI server when listening for traffic", err)
		}
	}()
	return nil
}

// unpackWebDirectory reads the web directory embedded into the binary, and copies it all to the node's root folder
func unpackWebDirectory() error {
	return fs.WalkDir(EmbeddedAssets, "build", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		webPath := filepath.Join(setting.Config.WebServer.Path, strings.TrimPrefix(path, "build"))

		_, err = os.Stat(webPath)
		if err == nil || !os.IsNotExist(err) {
			return err
		}

		contents, err := EmbeddedAssets.ReadFile(path)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Dir(webPath), 0700)
		if err != nil {
			return err
		}
		return os.WriteFile(webPath, contents, 0644)
	})
}
