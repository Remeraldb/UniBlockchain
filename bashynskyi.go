package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"blockchain/storage" // use your module name
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file")
	}

	rpcURL := os.Getenv("RPC_URL")
	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	privKeyHex := os.Getenv("PRIVATE_KEY")
	if rpcURL == "" || contractAddr == "" || privKeyHex == "" {
		log.Fatal("Missing env vars")
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal("RPC connection error:", err)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		log.Fatal("Invalid private key:", err)
	}

	contractAddress := common.HexToAddress(contractAddr)
	instance, err := storage.NewStorage(contractAddress, client)
	if err != nil {
		log.Fatal("Contract instance error:", err)
	}

	// Read initial value
	initialData, err := instance.Data(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Read error:", err)
	}
	fmt.Println("Initial data:", initialData)

	// Prepare transaction
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("Nonce error:", err)
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("Gas price error:", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal("ChainID error:", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatal("Auth error:", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasPrice = gasPrice
	auth.GasLimit = 300000

	// Send transaction
	newValue := "Hello from Go!"
	tx, err := instance.SetData(auth, newValue)
	if err != nil {
		log.Fatal("Transaction error:", err)
	}
	fmt.Println("Tx sent:", tx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatal("Mining error:", err)
	}
	if receipt.Status == 0 {
		log.Fatal("Transaction failed")
	}
	fmt.Println("Transaction confirmed")

	// Read new value
	newData, err := instance.Data(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Read new data error:", err)
	}
	fmt.Println("New data:", newData)

	// Listen to event
	blockNum := receipt.BlockNumber.Uint64() // convert *big.Int to uint64
	filterOpts := &bind.FilterOpts{
		Start: blockNum,
		End:   &blockNum,
	}
	iter, err := instance.FilterDataChanged(filterOpts)
	if err != nil {
		log.Fatal("Filter events error:", err)
	}
	for iter.Next() {
		event := iter.Event
		fmt.Printf("Event DataChanged: old=%s, new=%s\n", event.OldValue, event.NewValue)
	}
}
