package ipfs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/kubo/repo/fsrepo"
)

const TEMP_FOLDER = "tmp"

const IPFS_FOLDER = "ipfs"

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func createTempRepoFolder() (string, error) {
	repoPath, err := os.MkdirTemp(filepath.Join(setting.GetRootPath(), TEMP_FOLDER), "ipfs-shell")
	if err != nil {
		return "", errors.Wrap(err, "failed to get temp dir")
	}
	return repoPath, nil
}

func createTempRepo(repoPath string) (string, error) {
	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to init ephemeral node")
	}

	return repoPath, nil
}

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online: true,
		//Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo:    repo,
	}

	return core.NewNode(ctx, nodeOptions)
}

var loadPluginsOnce sync.Once

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context, repoPath string) (icore.CoreAPI, *core.IpfsNode, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, nil, onceErr
	}

	repoPath, err := createTempRepo(repoPath)
	if err != nil {
		return nil, nil, err
	}
	node, err := createNode(ctx, repoPath)
	if err != nil {
		return nil, nil, err
	}

	api, err := coreapi.NewCoreAPI(node)

	return api, node, err
}

// GetFile get ipfs file by spawning ephemeral node
func GetFile(ctx context.Context, cid string, fileName string) (string, error) {

	pp.Log(ctx, "-- Getting an IPFS node running -- ")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Spawn a node using a temporary path, creating a temporary repo for the run
	fmt.Println("Spawning Kubo node on a temporary repo")

	// Create a Temporary Repo
	repoPath, err := createTempRepoFolder()
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp repo folder")
	}
	defer os.RemoveAll(repoPath)

	ipfs, node, err := spawnEphemeral(ctx, repoPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to spawn ephemeral node")
	}
	defer node.Close()

	pp.Log(ctx, "IPFS node is running")

	pp.Log(ctx, "-- getting back files --")

	cidFile := icorepath.New(cid)
	outputBasePath := getTmpFileFolderPath(cid)
	err = os.MkdirAll(outputBasePath, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "could not create output dir")
	}
	pp.Logf(ctx, "output folder: %s", outputBasePath)
	outputPathFile := outputBasePath
	if fileName != "" {
		outputPathFile = path.Join(outputPathFile, fileName)
	} else {
		outputPathFile = path.Join(outputPathFile, strings.Split(cidFile.String(), "/")[2])
	}

	os.RemoveAll(outputPathFile)
	rootNodeFile, err := ipfs.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return "", errors.Wrap(err, "could not get file with CID")
	}

	err = files.WriteTo(rootNodeFile, outputPathFile)
	if err != nil {
		return "", errors.Wrap(err, "could not write out the fetched CID")
	}

	pp.Logf(ctx, "got file back from IPFS (IPFS path: %s) and wrote it to %s", cidFile.String(), outputPathFile)
	return outputPathFile, nil
}

func getTmpFileFolderPath(cid string) string {
	return filepath.Join(setting.GetRootPath(), TEMP_FOLDER, IPFS_FOLDER, cid)
}

// GetFileViaKuboCli get file from ipfs via kubo cli
func GetFileViaKuboCli(ctx context.Context, cid string, fileName string) (string, error) {
	outputBasePath := getTmpFileFolderPath(cid)
	err := os.MkdirAll(outputBasePath, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "could not create output dir")
	}
	outputPath := outputBasePath
	if fileName != "" {
		outputPath = path.Join(outputPath, fileName)
	} else {
		outputPath = path.Join(outputPath, cid)
	}
	transformCmd := exec.Command("ipfs", "get", cid, "-o", outputPath)
	stderr, _ := transformCmd.StderrPipe()
	if err = transformCmd.Start(); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		pp.Log(ctx, m)
	}
	if err = transformCmd.Wait(); err != nil {
		return "", err
	}
	return outputPath, nil
}
