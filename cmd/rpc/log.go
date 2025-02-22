package rpc

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"

	"github.com/canopy-network/canopy/lib"
	"github.com/julienschmidt/httprouter"
)

// logHandler allows debugging of incoming rpc calls by logging the inbound calls
type logHandler struct {
	path string
	h    httprouter.Handle
}

func (h logHandler) Handle(resp http.ResponseWriter, req *http.Request, p httprouter.Params) {
	//logger.Debug(h.path) can enable for developer debugging
	h.h(resp, req, p)
}

func logsHandler(s *Server) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		filePath := filepath.Join(s.config.DataDirPath, lib.LogDirectory, lib.LogFileName)
		f, _ := os.ReadFile(filePath)
		split := bytes.Split(f, []byte("\n"))
		var flipped []byte
		for i := len(split) - 1; i >= 0; i-- {
			flipped = append(append(flipped, split[i]...), []byte("\n")...)
		}
		if _, err := w.Write(flipped); err != nil {
			logger.Error(err.Error())
		}
	}
}
