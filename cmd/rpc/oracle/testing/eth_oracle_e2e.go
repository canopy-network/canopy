package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	erc20TransferMethodID = "a9059cbb"
	lockInterval          = 10 * time.Second

	chainId = 2
)

// TestCase represents a single test case with expected balance changes
type TestCase struct {
	Name                     string
	OrderAmount              uint64
	ExpectedUSDCTransfer     uint64
	ExpectedCNPYTransfer     uint64
	BuyerAddress             string
	BuyerPrivateKey          string
	SellerAddress            string
	SellerPrivateKey         string
	CanopyReceiveAddress     string
	CanopySendAddress        string
	InitialBuyerUSDCBalance  *big.Int
	InitialSellerUSDCBalance *big.Int
	InitialCNPYBalance       uint64
	OrderID                  string
	Status                   string // "created", "locked", "closed", "verified"
	Error                    error
}

// TestResults holds the results of all test cases
type TestResults struct {
	mutex     sync.RWMutex
	testCases map[string]*TestCase
	passed    int
	failed    int
	total     int
}

// All available Ethereum accounts from Anvil
var ethAccounts = [10]string{
	"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // Account 0
	"0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // Account 1
	"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", // Account 2
}

// Corresponding private keys for the accounts
var ethPrivateKeys = [10]string{
	"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Account 0
	"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", // Account 1
	"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", // Account 2
}

// Canopy accounts for receiving funds
var canopyAccounts = [2]string{
	"02cd4e5eb53ea665702042a6ed6d31d616054dc5",
	"851e90eaef1fa27debaee2c2591503bdeec1d123",
}

func main() {
	// Command line flags
	createOrder := flag.Bool("create-order", false, "Create a new sell order")
	lockOrder := flag.String("lock-order", "", "Lock an order by order ID")
	closeOrder := flag.String("close-order", "", "Close an order by order ID")
	runTests := flag.Bool("run-tests", false, "Run the full E2E test suite")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	// Order parameters
	amount := flag.Uint64("amount", 1000000, "Order amount in smallest unit (default: 1 USDC = 1000000)")
	buyerAddr := flag.String("buyer-addr", ethAccounts[0], "Buyer Ethereum address")
	buyerKey := flag.String("buyer-key", ethPrivateKeys[0], "Buyer private key")
	sellerAddr := flag.String("seller-addr", ethAccounts[1], "Seller Ethereum address")
	_ = flag.String("seller-key", ethPrivateKeys[1], "Seller private key") // Reserved for future use
	canopyAddr := flag.String("canopy-addr", canopyAccounts[0], "Canopy receive address")

	flag.Parse()

	// Show help if no flags provided
	if !*createOrder && *lockOrder == "" && *closeOrder == "" && !*runTests {
		fmt.Println("Usage:")
		fmt.Println("  --create-order                    Create a new sell order")
		fmt.Println("  --lock-order <order-id|first>     Lock an order (use 'first' for first unlocked)")
		fmt.Println("  --close-order <order-id|first>    Close an order (use 'first' for first locked)")
		fmt.Println("  --run-tests                       Run full E2E test suite")
		fmt.Println("  --verbose                         Enable verbose logging")
		fmt.Println("\nExamples:")
		fmt.Println("  ./eth_oracle_e2e --create-order")
		fmt.Println("  ./eth_oracle_e2e --lock-order first")
		fmt.Println("  ./eth_oracle_e2e --close-order first")
		fmt.Println("  ./eth_oracle_e2e --lock-order abc123def456")
		fmt.Println("\nOrder Parameters (all have defaults):")
		fmt.Printf("  --amount <amount>                 Order amount (default: 1000000)\n")
		fmt.Printf("  --buyer-addr <address>            Buyer address (default: %s)\n", ethAccounts[0])
		fmt.Printf("  --buyer-key <private-key>         Buyer private key (default: %s)\n", ethPrivateKeys[0])
		fmt.Printf("  --seller-addr <address>           Seller address (default: %s)\n", ethAccounts[1])
		fmt.Printf("  --seller-key <private-key>        Seller private key (default: %s)\n", ethPrivateKeys[1])
		fmt.Printf("  --canopy-addr <address>           Canopy address (default: %s)\n", canopyAccounts[0])
		return
	}

	dataDir := lib.DefaultDataDirPath()
	configFilePath := filepath.Join(dataDir, lib.ConfigFilePath)

	// load the config object
	c, err := lib.NewConfigFromFile(configFilePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	c.DataDirPath = dataDir

	e2e, err := NewEthOracleE2E(c, dataDir)
	if err != nil {
		fmt.Printf("Error initializing E2E tester: %v\n", err)
		return
	}

	// Route to appropriate operation
	if *createOrder {
		// Use default seller address if not provided or use first account
		sellerAddress := *sellerAddr
		if sellerAddress == "" {
			sellerAddress = ethAccounts[0]
		}

		// Use default canopy address if not provided
		canopyAddress := *canopyAddr
		if canopyAddress == "" {
			canopyAddress = canopyAccounts[0]
		}

		err := e2e.CreateSellOrder(*amount, *amount, sellerAddress, canopyAddress)
		if err != nil {
			fmt.Printf("Error creating order: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Order created successfully: %d CNPY -> %d USDC (seller: %s)\n", *amount, *amount, sellerAddress)
	} else if *lockOrder != "" {
		if *lockOrder == "first" || *lockOrder == "auto" {
			// Lock the first available unlocked order
			err := e2e.LockFirstOrder(*buyerAddr, *buyerKey, *canopyAddr)
			if err != nil {
				fmt.Printf("Error locking first available order: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("First available order locked successfully\n")
		} else {
			// Lock specific order by ID
			err := e2e.LockOrder(*lockOrder, *buyerAddr, *buyerKey, *canopyAddr)
			if err != nil {
				fmt.Printf("Error locking order: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Order %s locked successfully\n", *lockOrder)
		}
	} else if *closeOrder != "" {
		if *closeOrder == "first" || *closeOrder == "auto" {
			// Close the first available locked order
			err := e2e.CloseFirstOrder(*buyerKey, *amount)
			if err != nil {
				fmt.Printf("Error closing first available order: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("First available order closed successfully\n")
		} else {
			// Close specific order by ID
			err := e2e.CloseOrder(*closeOrder, *buyerKey, *amount)
			if err != nil {
				fmt.Printf("Error closing order: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Order %s closed successfully\n", *closeOrder)
		}
	} else if *runTests {
		if *verbose {
			fmt.Println("Running test suite in verbose mode")
		}
		e2e.RunTestSuite()
	} else {
		fmt.Println("Usage:")
		fmt.Println("  --create-order                     Create a new sell order")
		fmt.Println("  --lock-order <order-id>           Lock an existing order")
		fmt.Println("  --close-order <order-id>          Close a locked order")
		fmt.Println("  --run-tests                       Run full E2E test suite")
		fmt.Println("  --verbose                         Enable verbose logging")
		fmt.Println("\nOrder Parameters:")
		fmt.Println("  --amount <amount>                 Order amount (default: 1000000)")
		fmt.Println("  --buyer-addr <address>            Buyer Ethereum address")
		fmt.Println("  --buyer-key <private-key>         Buyer private key")
		fmt.Println("  --seller-addr <address>           Seller Ethereum address")
		fmt.Println("  --seller-key <private-key>        Seller private key")
		fmt.Println("  --canopy-addr <address>           Canopy receive address")
		os.Exit(0)
	}
}

// EthOracleE2E handles RPC requests to the canopy blockchain
type EthOracleE2E struct {
	ethClient   *ethclient.Client
	client      *rpc.Client
	dataDir     string
	logger      lib.LoggerI
	config      lib.Config
	testResults *TestResults
}

// NewEthOracleE2E creates a new E2E tester instance
func NewEthOracleE2E(config lib.Config, dataDir string) (*EthOracleE2E, error) {
	ethUrl := os.Getenv("ETH_RPC_URL")
	if ethUrl == "" {
		return nil, fmt.Errorf("ETH_RPC_URL environment variable not set")
	}

	// connect to rpc endpoint
	ethClient, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, err
	}

	// initialize logger
	logger := lib.NewDefaultLogger()

	// create client
	client := rpc.NewClient(config.RPCUrl, config.AdminRPCUrl)
	fmt.Println(config)

	return &EthOracleE2E{
		ethClient: ethClient,
		client:    client,
		dataDir:   dataDir,
		logger:    logger,
		config:    config,
		testResults: &TestResults{
			testCases: make(map[string]*TestCase),
		},
	}, nil
}

// RunTestSuite runs the complete test suite
func (e *EthOracleE2E) RunTestSuite() {
	e.logger.Info("Starting E2E Oracle Test Suite")

	// Delete all existing orders before starting tests
	err := e.deleteAllExistingOrders()
	if err != nil {
		e.logger.Errorf("Failed to delete existing orders: %v", err)
		return
	}

	// Generate test cases
	testCases := e.generateTestCases()

	// Run tests
	for _, testCase := range testCases {
		e.testResults.mutex.Lock()
		e.testResults.testCases[testCase.Name] = testCase
		e.testResults.total++
		e.testResults.mutex.Unlock()

		e.logger.Infof("Test %s - Started", testCase.Name)
		e.runTestCase(testCase)
	}

	// Wait for all tests to complete
	e.waitForTestCompletion()

	// Print final results
	e.printTestResults()
}

// generateTestCases creates test cases for different scenarios
func (e *EthOracleE2E) generateTestCases() []*TestCase {
	testCases := []*TestCase{
		{
			Name:                 "BasicOrderFlow_1000USDC",
			OrderAmount:          1000000, // 1 USDC in 6 decimals
			ExpectedUSDCTransfer: 1000000,
			ExpectedCNPYTransfer: 1000000,
			BuyerAddress:         ethAccounts[0],
			BuyerPrivateKey:      ethPrivateKeys[0],
			SellerAddress:        ethAccounts[1],
			SellerPrivateKey:     ethPrivateKeys[1],
			CanopyReceiveAddress: canopyAccounts[1],
			CanopySendAddress:    canopyAccounts[1],
			Status:               "created",
		},
		// {
		// 	Name:                 "LargeOrderFlow_10000USDC",
		// 	OrderAmount:          10000000, // 10 USDC in 6 decimals
		// 	ExpectedUSDCTransfer: 10000000,
		// 	ExpectedCNPYTransfer: 10000000,
		// 	BuyerAddress:         ethAccounts[1],
		// 	BuyerPrivateKey:      ethPrivateKeys[1],
		// 	SellerAddress:        ethAccounts[2],
		// 	SellerPrivateKey:     ethPrivateKeys[2],
		// 	CanopyReceiveAddress: canopyAccounts[1],
		// 	CanopySendAddress:    canopyAccounts[1],
		// 	Status:               "created",
		// },
		// {
		// 	Name:                 "BasicOrderFlow_1000USDC",
		// 	OrderAmount:          1000000, // 1 USDC in 6 decimals
		// 	ExpectedUSDCTransfer: 1000000,
		// 	ExpectedCNPYTransfer: 1000000,
		// 	BuyerAddress:         ethAccounts[0],
		// 	BuyerPrivateKey:      ethPrivateKeys[0],
		// 	SellerAddress:        ethAccounts[1],
		// 	SellerPrivateKey:     ethPrivateKeys[1],
		// 	CanopyReceiveAddress: canopyAccounts[1],
		// 	CanopySendAddress:    canopyAccounts[1],
		// 	Status:               "created",
		// },
		// {
		// 	Name:                 "LargeOrderFlow_10000USDC",
		// 	OrderAmount:          10000000, // 10 USDC in 6 decimals
		// 	ExpectedUSDCTransfer: 10000000,
		// 	ExpectedCNPYTransfer: 10000000,
		// 	BuyerAddress:         ethAccounts[1],
		// 	BuyerPrivateKey:      ethPrivateKeys[1],
		// 	SellerAddress:        ethAccounts[2],
		// 	SellerPrivateKey:     ethPrivateKeys[2],
		// 	CanopyReceiveAddress: canopyAccounts[1],
		// 	CanopySendAddress:    canopyAccounts[1],
		// 	Status:               "created",
		// },
	}

	return testCases
}

// runTestCase executes a single test case
func (e *EthOracleE2E) runTestCase(testCase *TestCase) {
	// Record initial balances
	e.recordInitialBalances(testCase)

	// Create order
	err := e.createTestOrder(testCase)
	if err != nil {
		e.failTestCase(testCase, fmt.Errorf("failed to create order: %w", err))
		return
	}

	// Wait for order to be available and lock it
	err = e.waitAndLockOrder(testCase)
	if err != nil {
		e.failTestCase(testCase, fmt.Errorf("failed to lock order: %w", err))
		return
	}

	// Close the order
	err = e.closeTestOrder(testCase)
	if err != nil {
		e.failTestCase(testCase, fmt.Errorf("failed to close order: %w", err))
		return
	}

	// Wait for order to be completed and removed from order book
	err = e.waitForOrderCompletion(testCase)
	if err != nil {
		e.failTestCase(testCase, fmt.Errorf("failed to wait for order completion: %w", err))
		return
	}

	// Verify final balances
	err = e.verifyFinalBalances(testCase)
	if err != nil {
		e.failTestCase(testCase, fmt.Errorf("balance verification failed: %w", err))
		return
	}

	e.passTestCase(testCase)
}

// recordInitialBalances records the initial balances before the test
func (e *EthOracleE2E) recordInitialBalances(testCase *TestCase) {
	var err error

	// Record initial USDC balances
	testCase.InitialBuyerUSDCBalance, err = e.getUSDCBalance(testCase.BuyerAddress)
	if err != nil {
		e.logger.Errorf("Failed to get initial buyer USDC balance: %v", err)
		testCase.InitialBuyerUSDCBalance = big.NewInt(0)
	}

	testCase.InitialSellerUSDCBalance, err = e.getUSDCBalance(testCase.SellerAddress)
	if err != nil {
		e.logger.Errorf("Failed to get initial seller USDC balance: %v", err)
		testCase.InitialSellerUSDCBalance = big.NewInt(0)
	}

	// Record initial CNPY balance
	testCase.InitialCNPYBalance, err = e.getCNPYBalance(testCase.CanopyReceiveAddress)
	if err != nil {
		e.logger.Errorf("Failed to get initial CNPY balance: %v", err)
		testCase.InitialCNPYBalance = 0
	}

	e.logger.Infof("Test %s - Initial balances: Buyer USDC=%s, Seller USDC=%s, CNPY=%d",
		testCase.Name,
		e.formatUSDCBalance(testCase.InitialBuyerUSDCBalance),
		e.formatUSDCBalance(testCase.InitialSellerUSDCBalance),
		testCase.InitialCNPYBalance)
}

// getAuth gets credentials from the env
func getAuth() (rpc.AddrOrNickname, string) {
	nick := os.Getenv("E2E_FROM_NICK")
	pass := os.Getenv("E2E_FROM_PASS")
	if nick == "" || pass == "" {
		panic(fmt.Sprintf("%s %s\n", nick, pass))
	}

	return rpc.AddrOrNickname{Nickname: nick}, pass

}

// CreateSellOrder creates a sell order with specified parameters
func (e *EthOracleE2E) CreateSellOrder(sellAmount, receiveAmount uint64, sellerAddress, canopyAddress string) error {
	// load the keystore from file
	_, err := crypto.NewKeystoreFromFile(e.dataDir)
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	from, pass := getAuth()

	receiveAddress := strings.TrimPrefix(sellerAddress, "0x")
	submit := true
	optFee := uint64(100000)
	contract := strings.TrimPrefix(os.Getenv("USDC_CONTRACT"), "0x")
	data, err := lib.NewHexBytesFromString(contract)
	if err != nil {
		return fmt.Errorf("failed to create contract data: %w", err)
	}

	_, _, err = e.client.TxCreateOrder(from, sellAmount, receiveAmount, chainId, receiveAddress, pass, data, submit, optFee)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	e.logger.Infof("Created sell order: %d CNPY -> %d USDC (seller: %s)",
		sellAmount, receiveAmount, sellerAddress)
	
	// Print balances after creating order
	e.printAccountBalances("Balances After Creating Order")

	return nil
}

// createTestOrder creates an order for the test case
func (e *EthOracleE2E) createTestOrder(testCase *TestCase) error {
	return e.CreateSellOrder(testCase.OrderAmount, testCase.ExpectedUSDCTransfer, testCase.SellerAddress, testCase.CanopyReceiveAddress)
}

// LockOrder locks an order by its ID with specified buyer parameters
func (e *EthOracleE2E) LockOrder(orderID, buyerAddress, buyerPrivateKey, canopyAddress string) error {
	// Find the order by ID
	targetOrder, err := e.findOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("failed to find order %s: %w", orderID, err)
	}

	if targetOrder.BuyerSendAddress != nil {
		return fmt.Errorf("order %s is already locked", orderID)
	}

	return e.lockOrderInternal(targetOrder, buyerAddress, buyerPrivateKey, canopyAddress)
}

// LockFirstOrder locks the first available unlocked order
func (e *EthOracleE2E) LockFirstOrder(buyerAddress, buyerPrivateKey, canopyAddress string) error {
	// Find the first unlocked order
	targetOrder, err := e.findFirstUnlockedOrder()
	if err != nil {
		return fmt.Errorf("failed to find unlocked order: %w", err)
	}

	return e.lockOrderInternal(targetOrder, buyerAddress, buyerPrivateKey, canopyAddress)
}

// lockOrderInternal handles the actual locking logic
func (e *EthOracleE2E) lockOrderInternal(targetOrder *lib.SellOrder, buyerAddress, buyerPrivateKey, canopyAddress string) error {
	// Lock the order
	heightPtr, err := e.client.Height()
	if err != nil {
		return fmt.Errorf("failed to get height: %w", err)
	}
	height := *heightPtr + 5

	lockOrder := &lib.LockOrder{
		OrderId:             targetOrder.Id,
		BuyerSendAddress:    common.FromHex(buyerAddress),
		BuyerReceiveAddress: common.Hex2Bytes(canopyAddress),
		BuyerChainDeadline:  height,
		ChainId:             chainId,
	}

	data, er := json.Marshal(lockOrder)
	if er != nil {
		return fmt.Errorf("failed to marshal lock order: %w", er)
	}

	sendAddress := common.HexToAddress(strings.TrimPrefix(buyerAddress, "0x"))
	err2 := SendTransaction(e.ethClient, sendAddress, buyerPrivateKey, new(big.Int).SetUint64(0), data)
	if err2 != nil {
		return fmt.Errorf("failed to send lock transaction: %w", err2)
	}

	orderID := lib.BytesToString(targetOrder.Id)
	e.logger.Infof("Lock order sent for order %s by buyer %s", orderID, buyerAddress)
	
	// Print balances after locking order
	e.printAccountBalances("Balances After Locking Order")
	return nil
}

// findOrderByID finds an order by its ID in the order books
func (e *EthOracleE2E) findOrderByID(orderID string) (*lib.SellOrder, error) {
	orders, err := e.Orders()
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	for _, book := range orders.OrderBooks {
		for _, order := range book.Orders {
			if lib.BytesToString(order.Id) == orderID {
				return order, nil
			}
		}
	}

	return nil, fmt.Errorf("order %s not found", orderID)
}

// findFirstUnlockedOrder finds the first unlocked order in the order books
func (e *EthOracleE2E) findFirstUnlockedOrder() (*lib.SellOrder, error) {
	orders, err := e.Orders()
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	for _, book := range orders.OrderBooks {
		for _, order := range book.Orders {
			if order.BuyerSendAddress == nil { // unlocked
				return order, nil
			}
		}
	}

	return nil, fmt.Errorf("no unlocked orders found")
}

// findFirstLockedOrder finds the first locked order in the order books
func (e *EthOracleE2E) findFirstLockedOrder() (*lib.SellOrder, error) {
	orders, err := e.Orders()
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	for _, book := range orders.OrderBooks {
		for _, order := range book.Orders {
			if order.BuyerSendAddress != nil { // locked
				return order, nil
			}
		}
	}

	return nil, fmt.Errorf("no locked orders found")
}

// waitAndLockOrder waits for the order to appear and locks it
func (e *EthOracleE2E) waitAndLockOrder(testCase *TestCase) error {
	// Wait for order to appear in order book
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	orderFound := false
	for !orderFound {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for order to appear")
		case <-ticker.C:
			orders, err := e.Orders()
			if err != nil {
				continue
			}
			// e.logger.Infof("Checking %d order books", len(orders.OrderBooks))

			for _, book := range orders.OrderBooks {
				// Find our order (look for unlocked orders with matching amounts)
				for _, order := range book.Orders {
					if order.BuyerSendAddress == nil && // unlocked
						order.AmountForSale == testCase.OrderAmount &&
						order.RequestedAmount == testCase.ExpectedUSDCTransfer {
						testCase.Status = "created"
						testCase.OrderID = lib.BytesToString(order.Id)
						orderFound = true
						break
					}
				}
			}
		}
	}

	return e.LockOrder(testCase.OrderID, testCase.BuyerAddress, testCase.BuyerPrivateKey, testCase.CanopyReceiveAddress)
}

// CloseOrder closes a locked order by sending USDC transfer with close order data
func (e *EthOracleE2E) CloseOrder(orderID, buyerPrivateKey string, transferAmount uint64) error {
	// Find the locked order by ID
	lockedOrder, err := e.findOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("failed to find order %s: %w", orderID, err)
	}

	if lockedOrder.BuyerSendAddress == nil {
		return fmt.Errorf("order %s is not locked", orderID)
	}

	return e.closeOrderInternal(lockedOrder, buyerPrivateKey, transferAmount)
}

// CloseFirstOrder closes the first available locked order
func (e *EthOracleE2E) CloseFirstOrder(buyerPrivateKey string, transferAmount uint64) error {
	// Find the first locked order
	lockedOrder, err := e.findFirstLockedOrder()
	if err != nil {
		return fmt.Errorf("failed to find locked order: %w", err)
	}

	return e.closeOrderInternal(lockedOrder, buyerPrivateKey, transferAmount)
}

// closeOrderInternal handles the actual closing logic
func (e *EthOracleE2E) closeOrderInternal(lockedOrder *lib.SellOrder, buyerPrivateKey string, transferAmount uint64) error {
	// Send USDC to the locked order's seller send address
	usdcContract := common.HexToAddress(strings.TrimPrefix(os.Getenv("USDC_CONTRACT"), "0x"))
	sellerReceiveAddress := common.BytesToAddress(lockedOrder.SellerReceiveAddress)

	// Create USDC transfer transaction
	transferData := erc20TransferMethodID +
		hex.EncodeToString(common.LeftPadBytes(sellerReceiveAddress.Bytes(), 32)) +
		hex.EncodeToString(common.LeftPadBytes(new(big.Int).SetUint64(transferAmount).Bytes(), 32))

	transferDataBytes, err := hex.DecodeString(transferData)
	if err != nil {
		return fmt.Errorf("failed to decode transfer data: %w", err)
	}

	// Create CloseOrder struct and marshal it
	closeOrder := &lib.CloseOrder{
		OrderId:    lockedOrder.Id,
		ChainId:    lockedOrder.Committee,
		CloseOrder: true,
	}

	closeOrderBytes, err := json.Marshal(closeOrder)
	if err != nil {
		return fmt.Errorf("failed to marshal close order: %w", err)
	}

	// Append the close order bytes to the transfer data
	finalTransferData := append(transferDataBytes, closeOrderBytes...)

	err = SendTransaction(e.ethClient, usdcContract, buyerPrivateKey, new(big.Int).SetUint64(0), finalTransferData)
	if err != nil {
		return fmt.Errorf("failed to send USDC transfer: %w", err)
	}

	orderID := lib.BytesToString(lockedOrder.Id)
	e.logger.Infof("Close order sent for order %s with %d USDC transfer", orderID, transferAmount)
	return nil
}

func (e *EthOracleE2E) sendClose(lockedOrder *lib.SellOrder, testCase *TestCase) error {
	e.logger.Infof("Test %s - %x locked order found", testCase.Name, lockedOrder.Id)

	return e.CloseOrder(lib.BytesToString(lockedOrder.Id), testCase.BuyerPrivateKey, testCase.ExpectedUSDCTransfer)
}

func (e *EthOracleE2E) closeTestOrder(testCase *TestCase) error {
	// Wait for order to be locked
	timeout := time.After(180 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var closed = []string{}

	done := false
	for !done {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for order %s to be locked", testCase.OrderID)
		case <-ticker.C:
			orders, err := e.Orders()
			if err != nil {
				continue
			}

			// Find our locked order
			for _, order := range orders.OrderBooks[0].Orders {
				if order.BuyerSendAddress != nil && // locked
					order.AmountForSale == testCase.OrderAmount &&
					order.RequestedAmount == testCase.ExpectedUSDCTransfer {
					testCase.Status = "locked"
					var send = true
					for _, id := range closed {
						if testCase.OrderID == id {
							send = false
						}
					}
					if send {
						e.sendClose(order, testCase)
						closed = append(closed, testCase.OrderID)
					}
				}
			}
		}
	}
	return nil
}

// waitForOrderCompletion waits for the order to be removed from the order book, indicating successful completion
func (e *EthOracleE2E) waitForOrderCompletion(testCase *TestCase) error {
	e.logger.Infof("Test %s - %s waiting for completion", testCase.Name, testCase.OrderID)

	timeout := time.After(120 * time.Second)  // Longer timeout for order completion
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for order %s to be completed and removed", testCase.OrderID)
		case <-ticker.C:
			orders, err := e.Orders()
			if err != nil {
				e.logger.Warnf("Failed to query orders during completion wait: %v", err)
				continue
			}

			// Check if our order is still in the order book
			orderFound := false
		orderLoop:
			for _, orderBook := range orders.OrderBooks {
				for _, order := range orderBook.Orders {
					if lib.BytesToString(order.Id) == testCase.OrderID {
						orderFound = true
						break orderLoop
					}
				}
			}

			// If order is not found in order book, it means it was completed successfully
			if !orderFound {
				e.logger.Infof("Test %s - %s order successfully completed and removed from order book", testCase.Name, testCase.OrderID)
				testCase.Status = "closed"
				return nil
			}

			// e.logger.Debugf("Test %s - Order %s still in order book, waiting for completion...", testCase.Name, testCase.OrderID)
		}
	}
}

// verifyFinalBalances verifies that the balances changed as expected
func (e *EthOracleE2E) verifyFinalBalances(testCase *TestCase) error {
	// Wait a bit for balances to update
	time.Sleep(5 * time.Second)

	// Get final balances
	finalBuyerUSDC, err := e.getUSDCBalance(testCase.BuyerAddress)
	if err != nil {
		return fmt.Errorf("failed to get final buyer USDC balance: %w", err)
	}

	finalSellerUSDC, err := e.getUSDCBalance(testCase.SellerAddress)
	if err != nil {
		return fmt.Errorf("failed to get final seller USDC balance: %w", err)
	}

	finalCNPY, err := e.getCNPYBalance(testCase.CanopyReceiveAddress)
	if err != nil {
		return fmt.Errorf("failed to get final CNPY balance: %w", err)
	}

	// Calculate actual changes
	buyerUSDCChange := new(big.Int).Sub(finalBuyerUSDC, testCase.InitialBuyerUSDCBalance)
	sellerUSDCChange := new(big.Int).Sub(finalSellerUSDC, testCase.InitialSellerUSDCBalance)
	cnpyChange := finalCNPY - testCase.InitialCNPYBalance

	// Log the changes
	e.logger.Infof("Test %s - Balance changes: Buyer USDC=%s, Seller USDC=%s, CNPY=%d",
		testCase.Name,
		e.formatUSDCBalance(buyerUSDCChange),
		e.formatUSDCBalance(sellerUSDCChange),
		cnpyChange)

	// Verify expected changes
	expectedSellerChange := new(big.Int).SetUint64(testCase.ExpectedUSDCTransfer)
	expectedBuyerChange := new(big.Int).Neg(expectedSellerChange)
	expectedCNPYChange := testCase.ExpectedCNPYTransfer

	if buyerUSDCChange.Cmp(expectedBuyerChange) != 0 {
		return fmt.Errorf("buyer USDC change mismatch: expected %s, got %s",
			e.formatUSDCBalance(expectedBuyerChange),
			e.formatUSDCBalance(buyerUSDCChange))
	}

	if sellerUSDCChange.Cmp(expectedSellerChange) != 0 {
		return fmt.Errorf("seller USDC change mismatch: expected %s, got %s",
			e.formatUSDCBalance(expectedSellerChange),
			e.formatUSDCBalance(sellerUSDCChange))
	}

	if cnpyChange != expectedCNPYChange {
		return fmt.Errorf("CNPY change mismatch: expected %d, got %d",
			expectedCNPYChange, cnpyChange)
	}

	testCase.Status = "verified"
	return nil
}

// isCanopyAddress checks if an address is a canopy address (shorter format without 0x prefix)
func (e *EthOracleE2E) isCanopyAddress(address string) bool {
	// Canopy addresses are shorter and don't have 0x prefix
	// Ethereum addresses are 42 chars with 0x prefix, or 40 chars without
	if len(address) == 40 && !strings.HasPrefix(address, "0x") {
		return true
	}
	return false
}

// printAccountBalances prints the balances of all related accounts for debugging
func (e *EthOracleE2E) printAccountBalances(label string) {
	fmt.Printf("\n=== %s ===\n", label)
	
	// Print Ethereum account USDC balances
	for i, account := range ethAccounts {
		usdcBalance, err := e.getUSDCBalance(account)
		if err != nil {
			fmt.Printf("ETH Account %d (%s): USDC balance error: %v\n", i, account, err)
		} else {
			fmt.Printf("ETH Account %d (%s): USDC balance: %s\n", i, account, e.formatUSDCBalance(usdcBalance))
		}
	}
	
	// Print Canopy account CNPY balances
	for i, account := range canopyAccounts {
		cnpyBalance, err := e.getCNPYBalance(account)
		if err != nil {
			fmt.Printf("Canopy Account %d (%s): CNPY balance error: %v\n", i, account, err)
		} else {
			fmt.Printf("Canopy Account %d (%s): CNPY balance: %d\n", i, account, cnpyBalance)
		}
	}
	fmt.Println("===========================")
}

// Helper functions
func (e *EthOracleE2E) getUSDCBalance(address string) (*big.Int, error) {
	usdcContract := common.HexToAddress(strings.TrimPrefix(os.Getenv("USDC_CONTRACT"), "0x"))
	account := common.HexToAddress(strings.TrimPrefix(address, "0x"))

	// ERC20 balanceOf method signature
	balanceOfMethodID := "70a08231"
	data := balanceOfMethodID + hex.EncodeToString(common.LeftPadBytes(account.Bytes(), 32))

	callData, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode call data: %w", err)
	}

	result, err := e.ethClient.CallContract(context.Background(), ethereum.CallMsg{
		To:   &usdcContract,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	return new(big.Int).SetBytes(result), nil
}

func (e *EthOracleE2E) getCNPYBalance(address string) (uint64, error) {
	account, err := e.client.Account(0, address)
	if err != nil {
		return 0, fmt.Errorf("failed to get CNPY balance: %w", err)
	}
	return account.Amount, nil
}

func (e *EthOracleE2E) formatUSDCBalance(balance *big.Int) string {
	// USDC has 6 decimal places
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	quotient := new(big.Int).Div(balance, divisor)
	remainder := new(big.Int).Mod(balance, divisor)

	return fmt.Sprintf("%s.%06d USDC", quotient.String(), remainder.Uint64())
}

func (e *EthOracleE2E) Orders() (*lib.OrderBooks, error) {
	orders, err := e.client.Orders(0, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	return orders, nil
}

// deleteAllExistingOrders deletes all existing orders before starting tests
func (e *EthOracleE2E) deleteAllExistingOrders() error {
	e.logger.Info("Deleting all existing orders before starting tests...")

	// Get all existing orders
	orders, err := e.Orders()
	if err != nil {
		return fmt.Errorf("failed to get existing orders: %w", err)
	}

	from, pass := getAuth()

	deletedCount := 0
	// Delete each order
	for _, orderBook := range orders.OrderBooks {
		for _, order := range orderBook.Orders {
			// Delete the order using e.client.TxDeleteOrder
			orderId := lib.BytesToString(order.Id)

			e.logger.Infof("Deleting order %s created by %s", orderId, from)

			_, _, err := e.client.TxDeleteOrder(from, orderId, chainId, pass, true, 100000)
			if err != nil {
				e.logger.Errorf("Failed to delete order %s: %v", orderId, err)
				continue
			}

			deletedCount++
		}
	}

	if deletedCount > 0 {
		e.logger.Infof("Successfully deleted %d existing orders", deletedCount)
		// Wait a moment for the deletions to be processed
		time.Sleep(10 * time.Second)
	}

	return nil
}

func (e *EthOracleE2E) passTestCase(testCase *TestCase) {
	e.testResults.mutex.Lock()
	defer e.testResults.mutex.Unlock()

	e.testResults.passed++
	e.logger.Infof("Test %s - PASSED ✅", testCase.Name)
}

func (e *EthOracleE2E) failTestCase(testCase *TestCase, err error) {
	e.testResults.mutex.Lock()
	defer e.testResults.mutex.Unlock()

	testCase.Error = err
	e.testResults.failed++
	e.logger.Errorf("Test %s - FAILED ❌: %v", testCase.Name, err)
}

func (e *EthOracleE2E) waitForTestCompletion() {
	// Wait for all tests to complete
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			e.logger.Errorf("Timeout waiting for test completion")
			return
		case <-ticker.C:
			e.testResults.mutex.RLock()
			completed := e.testResults.passed + e.testResults.failed
			total := e.testResults.total
			e.testResults.mutex.RUnlock()

			if completed >= total {
				return
			}
		}
	}
}

func (e *EthOracleE2E) printTestResults() {
	e.testResults.mutex.RLock()
	defer e.testResults.mutex.RUnlock()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("E2E ORACLE TEST RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("Total Tests: %d\n", e.testResults.total)
	fmt.Printf("Passed: %d\n", e.testResults.passed)
	fmt.Printf("Failed: %d\n", e.testResults.failed)
	fmt.Printf("Success Rate: %.2f%%\n", float64(e.testResults.passed)/float64(e.testResults.total)*100)

	if e.testResults.failed > 0 {
		fmt.Println("\nFailed Tests:")
		for name, testCase := range e.testResults.testCases {
			if testCase.Error != nil {
				fmt.Printf("  - %s: %v\n", name, testCase.Error)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}
