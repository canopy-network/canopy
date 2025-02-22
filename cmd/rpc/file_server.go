package rpc

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/canopy-network/canopy/lib"
)

//go:embed all:web/explorer/out
var explorerFS embed.FS

//go:embed all:web/wallet/out
var walletFS embed.FS

const (
	walletStaticDir   = "web/wallet/out"
	explorerStaticDir = "web/explorer/out"
)

func runStaticFileServer(fileSys fs.FS, dir, port string, conf lib.Config) {
	distFS, err := fs.Sub(fileSys, dir)
	if err != nil {
		logger.Error(fmt.Sprintf("an error occurred running the static file server for %s: %s", dir, err.Error()))
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// serve `index.html` with dynamic config injection
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			filePath := path.Join(dir, "index.html")
			data, e := fileSys.Open(filePath)
			if e != nil {
				http.NotFound(w, r)
				return
			}
			defer data.Close()

			htmlBytes, e := fs.ReadFile(fileSys, filePath)
			if e != nil {
				http.NotFound(w, r)
				return
			}

			// inject the config into the HTML file
			injectedHTML := injectConfig(string(htmlBytes), conf)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(injectedHTML))
			return
		}

		// Serve other files as-is
		http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
	})
	go func() {
		logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux).Error())
	}()
}

// injectConfig() injects the config.json into the HTML file
func injectConfig(html string, config lib.Config) string {
	script := fmt.Sprintf(`<script>
		window.__CONFIG__ = {
            rpcURL: "%s:%s",
            adminRPCURL: "%s:%s",
            chainId: %d
        };
	</script>`, config.RPCUrl, config.RPCPort, config.RPCUrl, config.AdminPort, config.ChainId)

	// inject the script just before </head>
	return strings.Replace(html, "</head>", script+"</head>", 1)
}
