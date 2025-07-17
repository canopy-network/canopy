package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	"334253d564e03c3397e11cdbc588b692bf8e31e8",
	"334253d564e03c3397e11cdbc588b692bf8e31e8",
}

func main() {
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

	// Run the test suite
	e2e.RunTestSuite()
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
		{
			Name:                 "LargeOrderFlow_10000USDC",
			OrderAmount:          10000000, // 10 USDC in 6 decimals
			ExpectedUSDCTransfer: 10000000,
			ExpectedCNPYTransfer: 10000000,
			BuyerAddress:         ethAccounts[1],
			BuyerPrivateKey:      ethPrivateKeys[1],
			SellerAddress:        ethAccounts[2],
			SellerPrivateKey:     ethPrivateKeys[2],
			CanopyReceiveAddress: canopyAccounts[1],
			CanopySendAddress:    canopyAccounts[1],
			Status:               "created",
		},
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
		{
			Name:                 "LargeOrderFlow_10000USDC",
			OrderAmount:          10000000, // 10 USDC in 6 decimals
			ExpectedUSDCTransfer: 10000000,
			ExpectedCNPYTransfer: 10000000,
			BuyerAddress:         ethAccounts[1],
			BuyerPrivateKey:      ethPrivateKeys[1],
			SellerAddress:        ethAccounts[2],
			SellerPrivateKey:     ethPrivateKeys[2],
			CanopyReceiveAddress: canopyAccounts[1],
			CanopySendAddress:    canopyAccounts[1],
			Status:               "created",
		},
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

// createTestOrder creates an order for the test case
func (e *EthOracleE2E) createTestOrder(testCase *TestCase) error {
	// load the keystore from file
	_, err := crypto.NewKeystoreFromFile(e.dataDir)
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	from, pass := getAuth()

	sellAmount := testCase.OrderAmount
	receiveAmount := testCase.ExpectedUSDCTransfer
	receiveAddress := strings.TrimPrefix(testCase.SellerAddress, "0x")
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

	e.logger.Infof("Test %s - Sent TxCreateOrder: %d CNPY -> %d USDC",
		testCase.Name, sellAmount, receiveAmount)

	return nil
}

// waitAndLockOrder waits for the order to appear and locks it
func (e *EthOracleE2E) waitAndLockOrder(testCase *TestCase) error {
	// Wait for order to appear in order book
	var targetOrder *lib.SellOrder
	timeout := time.After(30 * time.Second)
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
			e.logger.Infof("Checking %d order books", len(orders.OrderBooks))

			for _, book := range orders.OrderBooks {
				// Find our order (look for unlocked orders with matching amounts)
				for _, order := range book.Orders {
					if order.BuyerSendAddress == nil && // unlocked
						order.AmountForSale == testCase.OrderAmount &&
						order.RequestedAmount == testCase.ExpectedUSDCTransfer {
						targetOrder = order
						testCase.Status = "created"
						testCase.OrderID = lib.BytesToString(order.Id)
						orderFound = true
						break
					}
				}
			}
		}
	}

	// Lock the order
	heightPtr, err := e.client.Height()
	if err != nil {
		return fmt.Errorf("failed to get height: %w", err)
	}
	height := *heightPtr + 20

	lockOrder := &lib.LockOrder{
		OrderId:             targetOrder.Id,
		BuyerSendAddress:    common.FromHex(testCase.BuyerAddress),
		BuyerReceiveAddress: common.Hex2Bytes(testCase.CanopyReceiveAddress),
		BuyerChainDeadline:  height,
		ChainId:             chainId,
	}

	data, rr := json.Marshal(lockOrder)
	if rr != nil {
		return fmt.Errorf("failed to marshal lock order: %w", err)
	}

	sendAddress := common.HexToAddress(strings.TrimPrefix(testCase.BuyerAddress, "0x"))
	rr = SendTransaction(e.ethClient, sendAddress, testCase.BuyerPrivateKey, new(big.Int).SetUint64(0), data)
	if rr != nil {
		return fmt.Errorf("failed to send lock transaction: %w", err)
	}

	e.logger.Infof("Test %s - %x unlocked sell order found, sent lock order", testCase.Name, targetOrder.Id)
	return nil
}

func (e *EthOracleE2E) closeTestOrder(testCase *TestCase) error {
	// Wait for order to be locked
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lockedOrder *lib.SellOrder
	found := false
	for !found {
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
					lockedOrder = order
					testCase.Status = "locked"
					found = true
					break
				}
			}
		}
	}

	// Send USDC to the locked order's seller send address
	usdcContract := common.HexToAddress(strings.TrimPrefix(os.Getenv("USDC_CONTRACT"), "0x"))
	sellerReceiveAddress := common.BytesToAddress(lockedOrder.SellerReceiveAddress)

	// Create USDC transfer transaction
	transferData := erc20TransferMethodID +
		hex.EncodeToString(common.LeftPadBytes(sellerReceiveAddress.Bytes(), 32)) +
		hex.EncodeToString(common.LeftPadBytes(new(big.Int).SetUint64(testCase.ExpectedUSDCTransfer).Bytes(), 32))

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

	err = SendTransaction(e.ethClient, usdcContract, testCase.BuyerPrivateKey, new(big.Int).SetUint64(0), finalTransferData)
	if err != nil {
		return fmt.Errorf("failed to send USDC transfer: %w", err)
	}

	e.logger.Infof("Test %s - %x locked sell order found, sent close order", testCase.Name, lockedOrder.Id)
	return nil
}

// waitForOrderCompletion waits for the order to be removed from the order book, indicating successful completion
func (e *EthOracleE2E) waitForOrderCompletion(testCase *TestCase) error {
	e.logger.Infof("Test %s - %s waiting for order to be completed and removed from order book", testCase.Name, testCase.OrderID)

	timeout := time.After(60 * time.Second)   // Longer timeout for order completion
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

			_, _, err := e.client.TxDeleteOrder(from, orderId, 2, pass, true, 100000)
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
