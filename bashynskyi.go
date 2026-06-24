package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// --- КОНСТАНТИ ТА НАЛАШТУВАННЯ ---
const (
	BashynskyiBirthMonth = "02"         // Місяць народження (для перевірки хешу)
	BashynskyiNonce      = 19022006     // ДеньМісяцьРік (для генезис-блоку)
	BashynskyiPrevHash   = "Bashynskyi" // Прізвище для генезис-блоку
)

// BashynskyiTransaction - структура транзакції
type BashynskyiTransaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount"`
	TXID      string `json:"txid"`
}

// BashynskyiBlock - структура окремого блоку
type BashynskyiBlock struct {
	Index        int                     `json:"index"`
	Timestamp    int64                   `json:"timestamp"`
	Transactions []BashynskyiTransaction `json:"transactions"`
	Proof        int                     `json:"proof"` // Nonce
	PreviousHash string                  `json:"previous_hash"`
	Hash         string                  `json:"hash"`
}

// BashynskyiBlockchain - структура ланцюга
type BashynskyiBlockchain struct {
	Chain               []BashynskyiBlock
	CurrentTransactions []BashynskyiTransaction
}

// --- МЕТОДИ ДЛЯ ТРАНЗАКЦІЙ ---

// NewBashynskyiTransaction створює транзакцію та генерує її ідентифікатор
func NewBashynskyiTransaction(sender, recipient string, amount int) BashynskyiTransaction {
	tx := BashynskyiTransaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	data := sender + recipient + strconv.Itoa(amount)
	hash := sha256.Sum256([]byte(data))
	tx.TXID = hex.EncodeToString(hash[:])
	return tx
}

// --- МЕТОДИ ДЛЯ БЛОКІВ ТА ХЕШУВАННЯ ---

// CalculateBashynskyiHash обчислює SHA-256 хеш блоку
func (b *BashynskyiBlock) CalculateBashynskyiHash() string {
	txBytes, _ := json.Marshal(b.Transactions)
	// Об'єднуємо всі дані блоку в один рядок для хешування
	record := strconv.Itoa(b.Index) +
		strconv.FormatInt(b.Timestamp, 10) +
		string(txBytes) +
		strconv.Itoa(b.Proof) +
		b.PreviousHash

	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// --- МЕТОДИ БЛОКЧЕЙНУ ---

// NewBashynskyiBlockchain ініціалізує блокчейн та створює генезис-блок
func NewBashynskyiBlockchain() *BashynskyiBlockchain {
	bc := &BashynskyiBlockchain{
		Chain:               []BashynskyiBlock{},
		CurrentTransactions: []BashynskyiTransaction{},
	}
	bc.createBashynskyiGenesis()
	return bc
}

// createBashynskyiGenesis створює перший блок (Genesis)
func (bc *BashynskyiBlockchain) createBashynskyiGenesis() {
	timestamp := time.Now().Unix()

	// Шукаємо такий timestamp, щоб при фіксованому Nonce (дата народження)
	// хеш закінчувався на місяць народження
	for {
		genesisBlock := BashynskyiBlock{
			Index:        0,
			Timestamp:    timestamp,
			Transactions: []BashynskyiTransaction{},
			Proof:        BashynskyiNonce,
			PreviousHash: BashynskyiPrevHash,
		}
		genesisBlock.Hash = genesisBlock.CalculateBashynskyiHash()

		if strings.HasSuffix(genesisBlock.Hash, BashynskyiBirthMonth) {
			bc.Chain = append(bc.Chain, genesisBlock)
			break
		}
		timestamp++ // Перебір часу для валідності умови в лабі
	}
}

// AddBashynskyiBlock додає новий блок у ланцюг
func (bc *BashynskyiBlockchain) AddBashynskyiBlock(transactions []BashynskyiTransaction) {
	prevBlock := bc.Chain[len(bc.Chain)-1]
	newBlock := BashynskyiBlock{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		Proof:        0,
		PreviousHash: prevBlock.Hash,
	}

	// Процес майнінгу: підбір Proof (Nonce) для отримання потрібного суфікса хешу
	for {
		newBlock.Hash = newBlock.CalculateBashynskyiHash()
		if strings.HasSuffix(newBlock.Hash, BashynskyiBirthMonth) {
			break
		}
		newBlock.Proof++
	}

	bc.Chain = append(bc.Chain, newBlock)
}

// --- ВІЗУАЛІЗАЦІЯ ТА ПЕРЕВІРКА ---

func (bc *BashynskyiBlockchain) PrintBlockchain() {
	for _, block := range bc.Chain {
		fmt.Printf("--- Block %d ---\n", block.Index)
		fmt.Printf("Timestamp: %d\n", block.Timestamp)
		fmt.Printf("Proof:     %d\n", block.Proof)
		fmt.Printf("Prev Hash: %s\n", block.PreviousHash)
		fmt.Printf("Hash:      %s\n", block.Hash)
		fmt.Printf("Valid:     %v (ends with %s)\n\n",
			strings.HasSuffix(block.Hash, BashynskyiBirthMonth), BashynskyiBirthMonth)
	}
}

func main() {
	// 1. Створення блокчейну (з генезис-блоком)
	blockchain := NewBashynskyiBlockchain()

	// 2. Створення та додавання транзакцій
	tx1 := NewBashynskyiTransaction("System", "Bashynskyi", 100)
	blockchain.CurrentTransactions = append(blockchain.CurrentTransactions, tx1)

	// 3. Майнінг нового блоку
	blockchain.AddBashynskyiBlock(blockchain.CurrentTransactions)
	blockchain.CurrentTransactions = []BashynskyiTransaction{} // очистка

	// 4. Вивід результату
	blockchain.PrintBlockchain()
}
