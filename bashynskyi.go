package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// ===================== CONFIGURATION =====================

const contractABI = `[
  {
    "inputs": [],
    "name": "increment",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "counter",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "getCounter",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

var (
	client      *ethclient.Client
	contractAbi abi.ABI
	address     common.Address
	privateKey  *ecdsa.PrivateKey
	chainID     *big.Int
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		log.Fatal("RPC_URL not set")
	}

	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		log.Fatal("CONTRACT_ADDRESS not set")
	}

	privKeyHex := os.Getenv("PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("PRIVATE_KEY not set")
	}

	chainIDStr := os.Getenv("CHAIN_ID")
	if chainIDStr == "" {
		chainIDStr = "11155111"
	}
	id := new(big.Int)
	id.SetString(chainIDStr, 10)
	chainID = id

	// Connect to RPC
	var err error
	client, err = ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal("Failed to connect to RPC:", err)
	}

	contractAbi, err = abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		log.Fatal("Failed to parse ABI:", err)
	}

	address = common.HexToAddress(contractAddr)

	cleanKey := strings.TrimPrefix(privKeyHex, "0x")
	privateKey, err = crypto.HexToECDSA(cleanKey)
	if err != nil {
		log.Fatal("Failed to load private key:", err)
	}
}

// ===================== GET /contract/counter =====================
func getCounterHandler(c *gin.Context) {
	data, err := contractAbi.Pack("counter")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ABI pack failed"})
		return
	}

	msg := ethereum.CallMsg{To: &address, Data: data}
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Contract call failed"})
		return
	}

	values, err := contractAbi.Unpack("counter", result)
	if err != nil || len(values) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unpack result"})
		return
	}
	counter := values[0].(*big.Int)

	c.JSON(http.StatusOK, gin.H{
		"counter": counter.String(),
	})
}

// ===================== POST /contract/increment =====================
func incrementHandler(c *gin.Context) {
	startTime := time.Now()

	// Get sender address from private key
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Fetch nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nonce error"})
		return
	}

	// Suggest gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gas price error"})
		return
	}

	// Pack increment call
	data, err := contractAbi.Pack("increment")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ABI pack failed"})
		return
	}

	// Create unsigned transaction
	tx := types.NewTransaction(nonce, address, big.NewInt(0), 500000, gasPrice, data)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Signing failed"})
		return
	}

	// Send transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Send tx failed"})
		return
	}

	txHash := signedTx.Hash().Hex()

	// Wait for receipt (timeout 2 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, client, signedTx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Transaction not confirmed within timeout",
			"txHash": txHash,
			"hint":   "Check status on Etherscan later",
		})
		return
	}

	confirmationTime := time.Since(startTime).Milliseconds()

	// Return full transaction evaluation
	c.JSON(http.StatusOK, gin.H{
		"txHash":         txHash,
		"status":         receipt.Status, // 1 = success, 0 = fail
		"gasUsed":        receipt.GasUsed,
		"blockNumber":    receipt.BlockNumber.String(),
		"confirmationMs": confirmationTime,
	})
}

func main() {
	r := gin.Default()

	r.GET("/contract/counter", getCounterHandler)
	r.POST("/contract/increment", incrementHandler)

	fmt.Println("Server running on http://localhost:8088")
	r.Run(":8088")
}
