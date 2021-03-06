// Package rsync adds file server and copying client to copy files
// between glow driver and agent.
package rsync

import (
	"encoding/hex"
	"hash/crc32"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chrislusf/glow/util"
)

type FileHash struct {
	fullPath string `json:"path,omitempty"`
	File     string `json:"file,omitempty"`
	Hash     uint32 `json:"hash,omitempty"`
}

type RsyncServer struct {
	Port           int
	ExecutableFile string
	RelatedFiles   []string

	fileHashes []FileHash
}

func NewRsyncServer(file string, relatedFiles []string) (*RsyncServer, error) {
	rs := &RsyncServer{
		ExecutableFile: file,
		RelatedFiles:   relatedFiles,
	}
	if fh, err := GenerateFileHash(file); err != nil {
		log.Printf("Failed1 to read %s: %v", file, err)
	} else {
		rs.fileHashes = append(rs.fileHashes, *fh)
	}
	for _, f := range rs.RelatedFiles {
		if fh, err := GenerateFileHash(f); err != nil {
			log.Printf("Failed2 to read %s: %v", f, err)
		} else {
			rs.fileHashes = append(rs.fileHashes, *fh)
		}
	}
	return rs, nil
}

func (rs *RsyncServer) ExecutableFileHash() string {
	if len(rs.fileHashes) == 0 {
		return ""
	}
	hash := rs.fileHashes[0].Hash
	src := make([]byte, 4)
	dest := make([]byte, 8)
	util.Uint32toBytes(src, hash)
	hex.Encode(dest, src)
	return string(dest)
}

func (rs *RsyncServer) listHandler(w http.ResponseWriter, r *http.Request) {
	util.Json(w, r, http.StatusAccepted, ListFileResult{rs.fileHashes})
}

func (rs *RsyncServer) fileHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Path[len("/file/"):]
	for _, fh := range rs.fileHashes {
		if fh.File == fileName {
			file, err := os.Open(fh.fullPath)
			if err != nil {
				log.Printf("Can not read file: %s", fh.fullPath)
				return
			}
			defer file.Close()
			http.ServeContent(w, r, fh.File, time.Now(), file)
			return
		}
	}
}

// go start a http server locally that will respond predictably to ranged requests
func (rs *RsyncServer) Start() {
	s := http.NewServeMux()
	s.HandleFunc("/list", rs.listHandler)
	s.HandleFunc("/file/", rs.fileHandler)

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	rs.Port = listener.Addr().(*net.TCPAddr).Port

	go func() {
		http.Serve(listener, s)
	}()
}

func GenerateFileHash(fileName string) (*FileHash, error) {

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil, err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	hasher := crc32.NewIEEE()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, err
	}
	crc := hasher.Sum32()

	return &FileHash{
		fullPath: fileName,
		File:     filepath.Base(fileName),
		Hash:     crc,
	}, nil
}
