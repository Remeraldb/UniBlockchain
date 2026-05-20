package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	rpcURL  = "https://ethereum-sepolia-rpc.publicnode.com"
	chainID = 11155111 // Sepolia
)

// ========================== КОМАНДА CREATE ==========================
func createAccount() {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal("Помилка генерації ключа:", err)
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hexutil.Encode(privateKeyBytes)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Помилка перетворення публічного ключа")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	fmt.Println("=== НОВИЙ АКАУНТ ===")
	fmt.Println("Адреса:", address.Hex())
	fmt.Println("Приватний ключ (hex):", privateKeyHex)
	fmt.Println("\n⚠️ Збережіть приватний ключ у безпечному місці!")
}

// ========================== КОМАНДА BALANCE ==========================
func getBalance(addressHex string) {
	if !common.IsHexAddress(addressHex) {
		log.Fatal("Неправильний формат адреси")
	}
	address := common.HexToAddress(addressHex)

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal("Помилка підключення до RPC:", err)
	}
	defer client.Close()

	ctx := context.Background()
	balanceWei, err := client.BalanceAt(ctx, address, nil)
	if err != nil {
		log.Fatal("Помилка отримання балансу:", err)
	}

	// конвертуємо wei → ETH
	balanceETH := new(big.Float).Quo(
		new(big.Float).SetInt(balanceWei),
		new(big.Float).SetFloat64(1e18),
	)
	fmt.Printf("Баланс %s: %.6f ETH\n", addressHex, balanceETH)
}

// ========================== ПІДПИС ПОВІДОМЛЕНЬ ==========================
func signMessage(privateKeyHex, message string) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatal("Неправильний приватний ключ:", err)
	}

	// Хешуємо повідомлення за стандартом Ethereum (EIP-191 префікс)
	data := []byte(message)
	hash := crypto.Keccak256Hash(data)

	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		log.Fatal("Помилка підпису:", err)
	}

	fmt.Println("Підписано повідомлення:", message)
	fmt.Println("Хеш (Keccak256):", hash.Hex())
	fmt.Println("Підпис (hex):", hexutil.Encode(signature))

	// Перевірка підпису
	pubKeyBytes, err := crypto.Ecrecover(hash.Bytes(), signature)
	if err != nil {
		log.Fatal("Помилка відновлення публічного ключа:", err)
	}
	recoveredPubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		log.Fatal("Помилка розбору публічного ключа:", err)
	}
	recoveredAddress := crypto.PubkeyToAddress(*recoveredPubKey)

	originalAddress := crypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))

	fmt.Println("Перевірка підпису:", recoveredAddress.Hex() == originalAddress.Hex())
}

// ========================== НАДСИЛАННЯ ETH ==========================
func sendTransaction(privateKeyHex, toAddressHex string, amountETH float64) {
	// Приватний ключ
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatal("Неправильний приватний ключ:", err)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	toAddress := common.HexToAddress(toAddressHex)

	// Підключення до RPC
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal("Помилка підключення:", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Отримуємо nonce (кількість вже надісланих транзакцій)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		log.Fatal("Помилка отримання nonce:", err)
	}

	// Конвертуємо ETH → Wei
	amountWei := new(big.Int).Mul(
		big.NewInt(int64(amountETH*1e18)),
		big.NewInt(1),
	)

	// Газові параметри (Sepolia)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal("Помилка отримання gas price:", err)
	}
	gasLimit := uint64(21000) // стандарт для переказу ETH

	// Створюємо транзакцію
	tx := types.NewTransaction(
		nonce,
		toAddress,
		amountWei,
		gasLimit,
		gasPrice,
		nil, // дані порожні (простий переказ)
	)

	// Підписуємо транзакцію з ланцюгом Sepolia
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
	if err != nil {
		log.Fatal("Помилка підпису транзакції:", err)
	}

	// Надсилаємо
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatal("Помилка відправки транзакції:", err)
	}

	txHash := signedTx.Hash().Hex()
	fmt.Println("✅ Транзакцію надіслано!")
	fmt.Println("Хеш транзакції:", txHash)
	fmt.Println("Переглянути на Etherscan:", "https://sepolia.etherscan.io/tx/"+txHash)
}

// ========================== ОСНОВНА CLI ==========================
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Використання:")
		fmt.Println("  go run main.go create")
		fmt.Println("  go run main.go balance <address>")
		fmt.Println("  go run main.go sign <private_key_hex> \"message\"")
		fmt.Println("  go run main.go send <private_key_hex> <to_address> <amount_in_ETH>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "create":
		createAccount()

	case "balance":
		if len(os.Args) != 3 {
			log.Fatal("Використання: balance <address>")
		}
		getBalance(os.Args[2])

	case "sign":
		if len(os.Args) != 4 {
			log.Fatal("Використання: sign <private_key_hex> \"message\"")
		}
		signMessage(os.Args[2], os.Args[3])

	case "send":
		if len(os.Args) != 5 {
			log.Fatal("Використання: send <private_key_hex> <to_address> <amount_in_ETH>")
		}
		amount, err := strconv.ParseFloat(os.Args[4], 64)
		if err != nil {
			log.Fatal("Неправильна сума ETH")
		}
		sendTransaction(os.Args[2], os.Args[3], amount)

	default:
		fmt.Println("Невідома команда:", command)
	}
}

// Додати в import "strconv" (для parseFloat)
