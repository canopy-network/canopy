package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var rootCmd = &cobra.Command{
	Use:   "canopy",
	Short: "the canopy blockchain software",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(rpc.SoftwareVersion)
	},
}

var (
	client, config, l     = &rpc.Client{}, lib.Config{}, lib.LoggerI(nil)
	DataDir, validatorKey = "", crypto.PrivateKeyI(nil)
)

func init() {
	flag.Parse()
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(adminCmd)
	rootCmd.AddCommand(autoCompleteCmd)
	autoCompleteCmd.AddCommand(generateCompleteCmd)
	autoCompleteCmd.AddCommand(autoCompleteInstallCmd)
	rootCmd.PersistentFlags().StringVar(&DataDir, "data-dir", lib.DefaultDataDirPath(), "custom data directory location")
	config, validatorKey = InitializeDataDirectory(DataDir, lib.NewDefaultLogger())
	l = lib.NewLogger(lib.LoggerConfig{
		Level:      config.GetLogLevel(),
		Structured: config.Structured,
		JSON:       config.JSON,
	})
	client = rpc.NewClient(config.RPCUrl, config.AdminRPCUrl)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the blockchain software",
	Run: func(cmd *cobra.Command, args []string) {
		Start()
	},
}

// Start() is the entrypoint of the application
func Start() {
	// start the validator TCP proxy (if configured)
	proxy := lib.NewValidatorTCPProxy(config.ValidatorTCPProxy, l)
	if err := proxy.Start(); err != nil {
		l.Fatal(err.Error())
	}
	// initialize and start the metrics server
	metrics := lib.NewMetricsServer(validatorKey.PublicKey().Address(), float64(config.ChainId), rpc.SoftwareVersion, config.MetricsConfig, l)
	// create a new database object from the config
	db, err := store.New(config, metrics, l)
	if err != nil {
		l.Fatal(err.Error())
	}
	// log the validator identity
	l.Infof("Using identity: Address: %s | PublicKey: %s",
		validatorKey.PublicKey().Address().String(), validatorKey.PublicKey().Address().String())
	// initialize the state machine
	sm, err := fsm.New(config, db, new(lib.Plugin), metrics, l)
	if err != nil {
		l.Fatal(err.Error())
	}
	// create a new instance of the application
	app, err := controller.New(sm, config, validatorKey, metrics, l)
	if err != nil {
		l.Fatal(err.Error())
	}
	// initialize the rpc server
	rpcServer := rpc.NewServer(app, config, l)
	// start the metrics server
	metrics.Start()
	// start the application
	app.Start()
	// start the rpc server
	rpcServer.Start()
	// block until a kill signal is received
	waitForKill()
	proxy.Stop()
	// gracefully stop the app
	app.Stop()
	// gracefully stop the metrics server
	metrics.Stop()
	// exit
	os.Exit(0)
}

// waitForKill() blocks until a kill signal is received
func waitForKill() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGABRT)
	// block until kill signal is received
	s := <-stop
	l.Infof("Exit command %s received", s)
}

func getFirstPassword(log lib.LoggerI) string {
	// allow flag config to skip initial password
	if pwd == "" {
		// get the password from the user
		log.Infof("Enter password for your new private key:")
		password, e := term.ReadPassword(int(os.Stdin.Fd()))
		if e != nil {
			log.Fatal(e.Error())
		}
		if password == nil {
			log.Infof("Password cannot be empty")
			return getFirstPassword(log)
		}
		return string(password)
	}

	return pwd
}

// InitializeDataDirectory() populates the data directory with configuration and data files if missing
func InitializeDataDirectory(dataDirPath string, log lib.LoggerI) (c lib.Config, privateValKey crypto.PrivateKeyI) {
	// make the data dir if missing
	if err := os.MkdirAll(dataDirPath, os.ModePerm); err != nil {
		log.Fatal(err.Error())
	}
	// make the config.json file if missing
	configFilePath := filepath.Join(dataDirPath, lib.ConfigFilePath)
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		log.Infof("Creating %s file", lib.ConfigFilePath)
		if err = lib.DefaultConfig().WriteToFile(configFilePath); err != nil {
			log.Fatal(err.Error())
		}
	}
	// make the private key file if missing
	privateValKeyPath := filepath.Join(dataDirPath, lib.ValKeyPath)
	if _, err := os.Stat(privateValKeyPath); errors.Is(err, os.ErrNotExist) {
		blsPrivateKey, _ := crypto.NewBLS12381PrivateKey()
		log.Infof("Creating %s file", lib.ValKeyPath)
		if err = crypto.PrivateKeyToFile(blsPrivateKey, privateValKeyPath); err != nil {
			log.Fatal(err.Error())
		}
		pwd = getFirstPassword(log)
		// allow flag config to skip initial nickname
		if nick == "" {
			// get nickname from the user
			log.Infof("Enter nickname for your new private key:")
			_, e := fmt.Scanln(&nick)
			if e != nil {
				log.Fatal(e.Error())
			}
		}
		// load the keystore from file
		k, e := crypto.NewKeystoreFromFile(dataDirPath)
		if e != nil {
			log.Fatal(e.Error())
		}
		// import the validator key
		address, e := k.ImportRaw(blsPrivateKey.Bytes(), pwd, crypto.ImportRawOpts{
			Nickname: nick,
		})
		if e != nil {
			log.Fatal(e.Error())
		}
		// save keystore to the file
		if e = k.SaveToFile(dataDirPath); e != nil {
			log.Fatal(e.Error())
		}
		log.Infof("Imported validator key %s to keystore", address)
	}
	// make the proposals.json file if missing
	if _, err := os.Stat(filepath.Join(dataDirPath, lib.ProposalsFilePath)); errors.Is(err, os.ErrNotExist) {
		log.Infof("Creating %s file", lib.ProposalsFilePath)
		// create an example proposal
		blsPrivateKey, _ := crypto.NewBLS12381PrivateKey()
		proposals := make(fsm.GovProposals)
		a, _ := lib.NewAny(&lib.StringWrapper{Value: "example"})
		tx, e := fsm.NewTransaction(blsPrivateKey, &fsm.MessageChangeParameter{
			ParameterSpace: fsm.ParamSpaceCons + "|" + fsm.ParamSpaceFee + "|" + fsm.ParamSpaceVal + "|" + fsm.ParamSpaceGov,
			ParameterKey:   fsm.ParamProtocolVersion,
			ParameterValue: a,
			StartHeight:    1,
			EndHeight:      1000,
			Signer:         []byte(strings.Repeat("F", 20)),
		}, 1, 1, 10000, 1, "example")
		if e != nil {
			log.Fatal(e.Error())
		}
		jsonBytes, e := lib.MarshalJSONIndent(tx)
		if e != nil {
			log.Fatal(e.Error())
		}
		if err = proposals.Add(jsonBytes, true); err != nil {
			log.Fatal(err.Error())
		}
		if err = proposals.SaveToFile(dataDirPath); err != nil {
			log.Fatal(err.Error())
		}
	}
	// load the private key object
	privateValKey, err := crypto.NewBLS12381PrivateKeyFromFile(privateValKeyPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	// make the poll.json file if missing
	if _, err = os.Stat(filepath.Join(dataDirPath, lib.PollsFilePath)); errors.Is(err, os.ErrNotExist) {
		log.Infof("Creating %s file", lib.PollsFilePath)
		// create an example poll
		examplePollHash := crypto.HashString([]byte("example"))
		polls := &fsm.ActivePolls{
			Polls: map[string]map[string]bool{
				examplePollHash: {privateValKey.PublicKey().Address().String(): true},
			},
			PollMeta: map[string]*fsm.StartPoll{
				examplePollHash: {
					StartPoll: examplePollHash,
					Url:       "https://forum.cnpy.network/something",
					EndHeight: 1000000000000,
				},
			},
		}
		if err = polls.SaveToFile(dataDirPath); err != nil {
			log.Fatal(err.Error())
		}
	}
	// create the genesis file if missing
	genesisFilePath := filepath.Join(dataDirPath, lib.GenesisFilePath)
	if _, err = os.Stat(genesisFilePath); errors.Is(err, os.ErrNotExist) {
		log.Infof("Creating %s file", lib.GenesisFilePath)
		WriteDefaultGenesisFile(privateValKey, genesisFilePath)
	}
	// load the config object
	c, err = lib.NewConfigFromFile(configFilePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	// set the data-directory
	c.DataDirPath = dataDirPath
	return
}

func WriteDefaultGenesisFile(validatorPrivateKey crypto.PrivateKeyI, genesisFilePath string) {
	consPubKey := validatorPrivateKey.PublicKey()
	addr := consPubKey.Address()
	j := &fsm.GenesisState{
		Time:     uint64(time.Now().UnixMicro()),
		Accounts: []*fsm.Account{{Address: addr.Bytes(), Amount: 1000000}},
		Validators: []*fsm.Validator{{
			Address:      addr.Bytes(),
			PublicKey:    consPubKey.Bytes(),
			Committees:   []uint64{lib.CanopyChainId},
			NetAddress:   "tcp://localhost",
			StakedAmount: 1000000000000,
			Output:       addr.Bytes(),
			Compound:     true,
		}},
		Params: fsm.DefaultParams(),
	}
	bz, _ := json.MarshalIndent(j, "", "  ")
	if err := os.WriteFile(genesisFilePath, bz, 0777); err != nil {
		panic(err)
	}
}

func writeToConsole(a any, err error) {
	if err != nil {
		l.Fatal(err.Error())
	}
	switch a.(type) {
	case int, uint32, uint64:
		p := message.NewPrinter(language.English)
		if _, err := p.Printf("%d\n", a); err != nil {
			l.Fatal(err.Error())
		}
	case string, *string:
		fmt.Println(a)
	default:
		s, err := lib.MarshalJSONIndentString(a)
		if err != nil {
			l.Fatal(err.Error())
		}
		fmt.Println(s)
	}
}

// AUTO COMPLETE CODE BELOW

var autoCompleteCmd = &cobra.Command{
	Use:   "auto-complete",
	Short: "auto-complete generation and installation (for zsh and bash)",
}

var autoCompleteInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "automatically installs shell completion",
	Run: func(cmd *cobra.Command, args []string) {
		shell := detectShell()
		if shell == "" {
			writeToConsole(nil, errors.New("can't detect shell (only zsh or bash is supported)"))
			return
		}
		completionScript, profileFile := "", ""

		switch shell {
		case "bash":
			profileFile = getBashProfile()
			completionScript = `
canopy auto-complete generate > ~/.canopy-completion.sh

# Ensure completion script is sourced only once
if ! grep -q 'source ~/.canopy-completion.sh' ` + profileFile + `; then
    echo 'source ~/.canopy-completion.sh' >> ` + profileFile + `
fi`
		case "zsh":
			profileFile = "~/.zshrc"
			completionScript = `
mkdir -p ~/.zsh/completions
canopy auto-complete generate > ~/.zsh/completions/_canopy

# Ensure fpath is set only once
if ! grep -q 'fpath=(~/.zsh/completions $fpath)' ` + profileFile + `; then
    echo 'fpath=(~/.zsh/completions $fpath)' >> ` + profileFile + `
fi

# Ensure compinit is set only once
if ! grep -q 'autoload -Uz compinit && compinit' ` + profileFile + `; then
    echo 'autoload -Uz compinit && compinit' >> ` + profileFile + `
fi`
		default:
			writeToConsole(nil, errors.New("unsupported shell (only zsh or bash is supported)"))
			return
		}
		writeToConsole(fmt.Sprintf("Installing completion for: %s", shell), nil)
		err := exec.Command("sh", "-c", completionScript).Run()
		if err != nil {
			writeToConsole(nil, fmt.Errorf("error setting up completion:, %s", err.Error()))
		} else {
			writeToConsole(fmt.Sprintf("Completion installed. Restart your shell or run `source %s%s", profileFile, "`"), nil)
		}
	},
}

var generateCompleteCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate completion script",
	Run: func(cmd *cobra.Command, args []string) {
		switch detectShell() {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		default:
			cmd.Println("Unsupported shell. Use: bash or zsh")
		}
	},
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "bash") {
		return "bash"
	} else if strings.Contains(shell, "zsh") {
		return "zsh"
	} else if strings.Contains(shell, "fish") {
		return "fish"
	}
	return ""
}

func getBashProfile() string {
	if _, err := os.Stat(os.Getenv("HOME") + "/.bashrc"); err == nil {
		return "~/.bashrc"
	}
	return "~/.bash_profile"
}
