package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// -------------------- СТРУКТУРИ --------------------

// Transaction представляє одну транзакцію
type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount"`
	TXID      string `json:"txid"`
}

// Block представляє один блок у блокчейні
type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	Proof        int           `json:"proof"`
	PreviousHash string        `json:"previous_hash"`
	Hash         string        `json:"hash"`
}

// Blockchain містить ланцюг блоків та Mempool
type Blockchain struct {
	Chain   []Block
	Mempool []Transaction
}

// -------------------- ДОПОМІЖНІ ФУНКЦІЇ --------------------

// generateTXID створює унікальний ідентифікатор транзакції
func generateTXID(sender, recipient string, amount int) string {
	data := sender + recipient + strconv.Itoa(amount)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// calculateHash обчислює хеш блоку
func (b *Block) calculateHash() string {
	txBytes, _ := json.Marshal(b.Transactions)
	record := strconv.Itoa(b.Index) +
		strconv.FormatInt(b.Timestamp, 10) +
		string(txBytes) +
		strconv.Itoa(b.Proof) +
		b.PreviousHash
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// validProof перевіряє, чи хеш починається з "0000"
func validProof(proof int, lastProof int, previousHash string, transactions []Transaction, timestamp int64) bool {
	guess := strconv.Itoa(lastProof) + strconv.Itoa(proof) + previousHash
	hash := sha256.Sum256([]byte(guess))
	hashStr := hex.EncodeToString(hash[:])
	return strings.HasPrefix(hashStr, "0000")
}

// proofOfWork знаходить proof (nonce), який задовольняє умову
func proofOfWork(lastProof int, previousHash string, transactions []Transaction, timestamp int64) int {
	proof := 0
	for !validProof(proof, lastProof, previousHash, transactions, timestamp) {
		proof++
	}
	return proof
}

// -------------------- МЕТОДИ БЛОКЧЕЙНУ --------------------

// NewBlockchain створює новий блокчейн із генезис-блоком
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		Chain:   []Block{},
		Mempool: []Transaction{},
	}
	bc.createGenesisBlock()
	return bc
}

// createGenesisBlock створює перший блок (індекс 0, previous_hash = "0")
func (bc *Blockchain) createGenesisBlock() {
	genesis := Block{
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{},
		Proof:        0,
		PreviousHash: "0",
	}
	genesis.Hash = genesis.calculateHash()
	bc.Chain = append(bc.Chain, genesis)
}

// addBlock створює новий блок із заданими транзакціями (виконує майнінг)
func (bc *Blockchain) addBlock(transactions []Transaction) {
	prevBlock := bc.Chain[len(bc.Chain)-1]
	newProof := proofOfWork(prevBlock.Proof, prevBlock.Hash, transactions, time.Now().Unix())

	newBlock := Block{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		Proof:        newProof,
		PreviousHash: prevBlock.Hash,
	}
	newBlock.Hash = newBlock.calculateHash()
	bc.Chain = append(bc.Chain, newBlock)
}

// -------------------- API-ОБРОБНИКИ (GIN) --------------------

// formatResponse уніфікує відповіді сервера
func formatResponse(status string, data interface{}) gin.H {
	return gin.H{
		"status": status,
		"data":   data,
	}
}

// getBlockchainHandler повертає весь ланцюг блоків
func getBlockchainHandler(c *gin.Context, bc *Blockchain) {
	c.JSON(http.StatusOK, formatResponse("success", bc.Chain))
}

// addTransactionHandler додає нову транзакцію в Mempool
func addTransactionHandler(c *gin.Context, bc *Blockchain) {
	var tx Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, formatResponse("error", "Invalid JSON"))
		return
	}

	// Генеруємо TXID
	tx.TXID = generateTXID(tx.Sender, tx.Recipient, tx.Amount)

	// Додаємо в Mempool
	bc.Mempool = append(bc.Mempool, tx)

	c.JSON(http.StatusCreated, formatResponse("success", gin.H{
		"message":     "Transaction added to mempool",
		"transaction": tx,
	}))
}

// mineBlockHandler виконує майнінг: бере всі транзакції з Mempool, створює блок, очищує Mempool
func mineBlockHandler(c *gin.Context, bc *Blockchain) {
	if len(bc.Mempool) == 0 {
		c.JSON(http.StatusBadRequest, formatResponse("error", "No transactions to mine"))
		return
	}

	// Копіюємо поточні транзакції з Mempool
	transactionsToMine := make([]Transaction, len(bc.Mempool))
	copy(transactionsToMine, bc.Mempool)

	// Додаємо новий блок із цими транзакціями
	bc.addBlock(transactionsToMine)

	// Очищуємо Mempool
	bc.Mempool = []Transaction{}

	// Отримуємо щойно доданий блок
	newBlock := bc.Chain[len(bc.Chain)-1]

	c.JSON(http.StatusOK, formatResponse("success", gin.H{
		"message":      "New block mined successfully",
		"block_index":  newBlock.Index,
		"block_hash":   newBlock.Hash,
		"proof":        newBlock.Proof,
		"transactions": newBlock.Transactions,
	}))
}

// -------------------- ГОЛОВНА ФУНКЦІЯ --------------------

func main() {
	// Ініціалізація блокчейну
	blockchain := NewBlockchain()

	// Створення маршрутизатора Gin
	r := gin.Default()

	// Ендпоінти API
	r.GET("/blockchain", func(c *gin.Context) {
		getBlockchainHandler(c, blockchain)
	})
	r.POST("/transaction", func(c *gin.Context) {
		addTransactionHandler(c, blockchain)
	})
	r.GET("/mine", func(c *gin.Context) {
		mineBlockHandler(c, blockchain)
	})

	// Запуск сервера на порту 8088
	fmt.Println("Server is running on http://localhost:8088")
	r.Run(":8088")
}
