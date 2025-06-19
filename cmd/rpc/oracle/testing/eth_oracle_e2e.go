package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/cmd/rpc/oracle/eth"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	ETH_ACCOUNT_0 = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	ETH_ACCOUNT_1 = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

	ETH_PRIVATE_KEY_0 = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	ETH_PRIVATE_KEY_1 = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

	CANOPY_ACCOUNT_1 = "851e90eaef1fa27debaee2c2591503bdeec1d123"
	CANOPY_ACCOUNT_2 = "6f600fd94290f5604e735a21074667dcfecef39c"

	erc20TransferMethodID = "a9059cbb"

	lockInterval = 30 * time.Second
)

func main() {
	dataDir := lib.DefaultDataDirPath()

	configFilePath := filepath.Join(dataDir, lib.ConfigFilePath)
	// load the config object
	c, err := lib.NewConfigFromFile(configFilePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	// set the data-directory
	c.DataDirPath = dataDir

	e2e, err := NewEthOracleE2E(c, dataDir)
	if err != nil {
		fmt.Printf("Error initializing generator: %v\n", err)
		return
	}

	e2e.Start()
	done := make(chan struct{})
	<-done
}

// EthOracleE2E handles RPC requests to the canopy blockchain
type EthOracleE2E struct {
	ethClient *ethclient.Client
	client    *rpc.Client
	dataDir   string
	logger    lib.LoggerI
	config    lib.Config

	lockTime  map[string]time.Time
	closeTime map[string]time.Time
}

// NewEthOracleE2E creates a new Generator instance
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
		lockTime:  make(map[string]time.Time),
		closeTime: make(map[string]time.Time),
	}, nil
}

func (g *EthOracleE2E) Start() {
	go g.MaintainOrders(1)
	go g.createLockOrders()
	go g.createCloseOrders()
}

// MaintainOrders ensures there is always a certain amount of orders in the order book
func (g *EthOracleE2E) MaintainOrders(amount int) {
	g.logger.Infof("Maintaining sell order count at %d...", amount)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// orders, err := g.Orders()
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// for _, order := range orders.OrderBooks[0].Orders {
	// 	g.DeleteOrder(lib.BytesToString(order.Id))
	// }

	for {
		<-ticker.C // Wait for the ticker to tick
		orders, err := g.Orders()
		if err != nil {
			fmt.Println(err)
			continue
		}
		numOrders := len(orders.OrderBooks[0].Orders)
		if numOrders >= amount {
			continue
		}
		g.logger.Infof("%d orders found, creating order", numOrders)

		_, err = g.CreateOrder()
		if err != nil {
			fmt.Printf("Error creating order: %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}

func (g *EthOracleE2E) DeleteOrder(orderId string) (any, error) {

	// load the keystore from file
	_, e := crypto.NewKeystoreFromFile(g.dataDir)
	if e != nil {
		log.Fatal(e.Error())
	}

	// create an order
	from := rpc.AddrOrNickname{Nickname: "localhost"}
	chainId := uint64(1)
	pwd := "test"
	submit := true
	optFee := uint64(100000)

	g.logger.Infof("Deleting order %s %s", from.Nickname, orderId)
	_, _, err := g.client.TxDeleteOrder(from, orderId, chainId, pwd, submit, optFee)
	if err != nil {
		g.logger.Errorf("failed to create order: %v", err)
		return nil, err
	}

	// log success
	// g.logger.Infof("successfully retrieved %d CreateOrder", len(ob.OrderBooks))
	return nil, nil
}

func (g *EthOracleE2E) CreateOrder() (any, error) {

	// load the keystore from file
	_, e := crypto.NewKeystoreFromFile(g.dataDir)
	if e != nil {
		log.Fatal(e.Error())
	}

	// create an order
	from := rpc.AddrOrNickname{Nickname: "localhost"}
	sellAmount := uint64(1000000)
	receiveAmount := uint64(1000000)
	chainId := uint64(1)
	receiveAddress := strings.TrimPrefix(ETH_ACCOUNT_0, "0x")
	pwd := "test"
	submit := true
	optFee := uint64(100000)
	contract := strings.TrimPrefix(os.Getenv("USDC_CONTRACT"), "0x")
	data, err := lib.NewHexBytesFromString(contract)
	if err != nil {
		return nil, err
	}

	g.logger.Infof("Creating order from nick %s for %d CNPY -> %d USDC, eth receive address %s", from.Nickname, sellAmount, receiveAmount, receiveAddress)
	_, _, err = g.client.TxCreateOrder(from, sellAmount, receiveAmount, chainId, receiveAddress, pwd, data, submit, optFee)
	if err != nil {
		g.logger.Errorf("failed to create order: %v", err)
		return nil, err
	}

	// log success
	// g.logger.Infof("successfully retrieved %d CreateOrder", len(ob.OrderBooks))
	return nil, nil
}

// Orders retrieves all orders
func (g *EthOracleE2E) Orders() (*lib.OrderBooks, error) {
	// send request to get orders
	ob, err := g.client.Orders(0, 1)
	if err != nil {
		g.logger.Errorf("failed to retrieve orders: %v", err)
		return nil, err
	}

	return ob, nil
}

// createLockOrders monitors the order book creating lock orders for all sell orders without one
func (g *EthOracleE2E) createLockOrders() {
	g.logger.Info("Watching for unlocked sell orders..")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		orders, err := g.Orders()
		if err != nil {
			fmt.Println(err)
			continue
		}
		ob := orders.OrderBooks[0]

		for _, order := range ob.Orders {
			// Skip locked orders
			if order.BuyerSendAddress != nil {
				continue
			}

			orderId := lib.BytesToString(order.Id)
			if t, ok := g.lockTime[orderId]; ok {
				if time.Since(t) < lockInterval {
					continue
				}
				delete(g.lockTime, orderId)
			}
			height, e := g.client.Height()
			if e != nil {
				g.logger.Errorf("Unable to create lock order: %v", e.Error())
			}
			*height += 60

			lockOrder := &lib.LockOrder{
				OrderId:             order.Id,
				BuyerSendAddress:    common.FromHex(ETH_ACCOUNT_1),
				BuyerReceiveAddress: common.Hex2Bytes(CANOPY_ACCOUNT_2),
				BuyerChainDeadline:  *height,
			}
			data, err := json.Marshal(lockOrder)
			if err != nil {
				continue
			}
			sendAddress := common.HexToAddress(strings.TrimPrefix(ETH_ACCOUNT_1, "0x"))
			g.logger.Infof("Found unlocked sell order %s, self sending from %s", lib.BytesToString(lockOrder.OrderId), sendAddress)
			err = eth.SendTransaction(g.ethClient,
				sendAddress,
				ETH_PRIVATE_KEY_1,
				new(big.Int).SetUint64(0),
				data)
			if err != nil {
				g.logger.Error(err.Error())
			}
			g.lockTime[orderId] = time.Now()
		}
	}
}

// createLockOrders monitors the order book creating lock orders for all sell orders without one
func (g *EthOracleE2E) createCloseOrders() {
	g.logger.Info("Watching for locked sell orders..")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C // Wait for the ticker to tick
		orders, err := g.Orders()
		if err != nil {
			fmt.Println(err)
			continue
		}
		ob := orders.OrderBooks[0]

		for _, order := range ob.Orders {
			// Skip unlocked orders
			if order.BuyerSendAddress == nil {
				continue
			}

			orderId := lib.BytesToString(order.Id)
			if t, ok := g.closeTime[orderId]; ok {
				if time.Since(t) < lockInterval {
					continue
				}
				delete(g.closeTime, orderId)
			}

			closeOrder := &lib.CloseOrder{
				OrderId:    order.Id,
				CloseOrder: true,
			}
			data, err := json.Marshal(closeOrder)
			if err != nil {
				continue
			}
			sellerAddress := lib.BytesToString(order.SellerReceiveAddress)
			g.logger.Infof("Found locked sell order %s, sending %d to %s, contract %s",
				lib.BytesToString(order.Id),
				order.RequestedAmount,
				sellerAddress,
				lib.BytesToString(order.Data),
			)

			contract := common.BytesToAddress(order.Data)

			erc20data := createERC20TransferData(order.SellerReceiveAddress, new(big.Int).SetUint64(order.RequestedAmount), data)

			err = eth.SendTransaction(
				g.ethClient,
				contract,
				ETH_PRIVATE_KEY_1,
				new(big.Int).SetUint64(0),
				erc20data)
			if err != nil {
				g.logger.Errorf("Error sending to ERC20 contract %x %v", fmt.Sprintf("%x", contract), err)
			}
		}
	}
}

func createERC20TransferData(recipientBytes []byte, amount *big.Int, extra []byte) []byte {
	// Initialize empty byte slice to build the transaction data
	data := make([]byte, 0)
	// Decode the ERC20 transfer method ID from hex string and append to data
	methodIDBytes, _ := hex.DecodeString(erc20TransferMethodID)
	data = append(data, methodIDBytes...)
	// Remove "0x" prefix from recipient address and decode from hex
	// Create 32-byte padded recipient address (addresses are 20 bytes, padded with 12 zero bytes at start)
	paddedRecipient := make([]byte, 32)
	copy(paddedRecipient[12:], recipientBytes)
	// Print recipient bytes for debugging (should be removed in production)
	// Append padded recipient address to transaction data
	data = append(data, paddedRecipient...)
	// Convert big.Int amount to bytes
	amountBytes := amount.Bytes()
	// Create 32-byte padded amount (right-aligned, padded with zeros at start)
	paddedAmount := make([]byte, 32)
	copy(paddedAmount[32-len(amountBytes):], amountBytes)
	// Append padded amount to transaction data
	data = append(data, paddedAmount...)
	// If extra data is provided, append it to the transaction data
	if extra != nil {
		data = append(data, extra...)
	}
	// Return the complete encoded transaction data
	return data
}
