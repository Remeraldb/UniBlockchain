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

// -------------------- СТРУКТУРИ РЕЄСТРУ ТА БЛОКЧЕЙНУ --------------------

// RegistryRecord – запис цифрового реєстру (предметна область: студенти)
type RegistryRecord struct {
	RecordID  string `json:"record_id"` // унікальний ідентифікатор запису
	FullName  string `json:"full_name"` // ПІБ студента
	Data      string `json:"data"`      // додаткова інформація (наприклад, спеціальність)
	Timestamp int64  `json:"timestamp"` // час створення запису
}

// Transaction – одиниця, що зберігається в Mempool та блоках
type Transaction struct {
	TXID   string         `json:"txid"`
	Record RegistryRecord `json:"record"`
}

// Block – структура блоку
type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	Proof        int           `json:"proof"`
	PreviousHash string        `json:"previous_hash"`
	Hash         string        `json:"hash"`
}

// Blockchain – основний об'єкт
type Blockchain struct {
	Chain   []Block
	Mempool []Transaction
}

// -------------------- ДОПОМІЖНІ ФУНКЦІЇ --------------------

// generateTXID створює унікальний ID транзакції на основі даних запису
func generateTXID(record RegistryRecord) string {
	data := record.RecordID + record.FullName + record.Data + strconv.FormatInt(record.Timestamp, 10)
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

// proofOfWork знаходить proof (nonce), який задовольняє умову PoW
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

// createGenesisBlock створює перший блок (порожній)
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

// addBlock додає новий блок (виконується під час майнінгу)
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

// getAllRecords – збирає всі RegistryRecord з усіх блоків
func (bc *Blockchain) getAllRecords() []RegistryRecord {
	var records []RegistryRecord
	for _, block := range bc.Chain {
		for _, tx := range block.Transactions {
			records = append(records, tx.Record)
		}
	}
	return records
}

// findRecordByID – перевіряє існування запису за RecordID
func (bc *Blockchain) findRecordByID(recordID string) *RegistryRecord {
	for _, block := range bc.Chain {
		for _, tx := range block.Transactions {
			if tx.Record.RecordID == recordID {
				return &tx.Record
			}
		}
	}
	return nil
}

// -------------------- API-ОБРОБНИКИ (GIN) --------------------

// formatResponse уніфікує відповіді
func formatResponse(status string, data interface{}) gin.H {
	return gin.H{
		"status": status,
		"data":   data,
	}
}

// 1. POST /registry/record – додати запис реєстру (створює транзакцію в Mempool)
func addRecordHandler(c *gin.Context, bc *Blockchain) {
	var record RegistryRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, formatResponse("error", "Invalid JSON"))
		return
	}
	// Якщо RecordID не задано клієнтом – згенеруємо унікальний (опційно)
	if record.RecordID == "" {
		record.RecordID = generateTXID(record) // простий спосіб
	}
	record.Timestamp = time.Now().Unix()

	// Створюємо транзакцію
	tx := Transaction{
		TXID:   generateTXID(record),
		Record: record,
	}
	// Додаємо в Mempool
	bc.Mempool = append(bc.Mempool, tx)

	c.JSON(http.StatusCreated, formatResponse("success", gin.H{
		"message": "Record added to mempool",
		"txid":    tx.TXID,
		"record":  record,
	}))
}

// 2. GET /registry/records – отримати всі записи (читання блокчейну)
func getRecordsHandler(c *gin.Context, bc *Blockchain) {
	records := bc.getAllRecords()
	c.JSON(http.StatusOK, formatResponse("success", records))
}

// 3. GET /registry/verify/:id – перевірити існування запису за RecordID
func verifyRecordHandler(c *gin.Context, bc *Blockchain) {
	recordID := c.Param("id")
	record := bc.findRecordByID(recordID)
	if record == nil {
		c.JSON(http.StatusNotFound, formatResponse("error", "Record not found"))
		return
	}
	c.JSON(http.StatusOK, formatResponse("success", record))
}

// 4. GET /mine – майнінг (взяття всіх транзакцій з Mempool, створення блоку)
func mineHandler(c *gin.Context, bc *Blockchain) {
	if len(bc.Mempool) == 0 {
		c.JSON(http.StatusBadRequest, formatResponse("error", "No transactions to mine"))
		return
	}
	// Копіюємо поточні транзакції
	transactionsToMine := make([]Transaction, len(bc.Mempool))
	copy(transactionsToMine, bc.Mempool)

	// Додаємо новий блок
	bc.addBlock(transactionsToMine)

	// Очищуємо Mempool
	bc.Mempool = []Transaction{}

	newBlock := bc.Chain[len(bc.Chain)-1]
	c.JSON(http.StatusOK, formatResponse("success", gin.H{
		"message":      "New block mined",
		"block_index":  newBlock.Index,
		"block_hash":   newBlock.Hash,
		"proof":        newBlock.Proof,
		"transactions": newBlock.Transactions,
	}))
}

// 5. GET /blockchain – отримати весь ланцюг (для налагодження)
func getBlockchainHandler(c *gin.Context, bc *Blockchain) {
	c.JSON(http.StatusOK, formatResponse("success", bc.Chain))
}

// -------------------- ГОЛОВНА ФУНКЦІЯ --------------------

func main() {
	// Ініціалізація блокчейну
	blockchain := NewBlockchain()

	// Створення маршрутизатора Gin
	r := gin.Default()

	// API цифрового реєстру
	r.POST("/registry/record", func(c *gin.Context) {
		addRecordHandler(c, blockchain)
	})
	r.GET("/registry/records", func(c *gin.Context) {
		getRecordsHandler(c, blockchain)
	})
	r.GET("/registry/verify/:id", func(c *gin.Context) {
		verifyRecordHandler(c, blockchain)
	})

	// Загальні API блокчейну (з лаб2)
	r.GET("/blockchain", func(c *gin.Context) {
		getBlockchainHandler(c, blockchain)
	})
	r.GET("/mine", func(c *gin.Context) {
		mineHandler(c, blockchain)
	})

	// Запуск сервера
	fmt.Println("Digital Registry API running on http://localhost:8088")
	r.Run(":8088")
}
