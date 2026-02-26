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

// Константа для місяця народження
const birthMonthSuffix = "02"

// BashynskyiTransaction представляє транзакцію
type BashynskyiTransaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount"`
	TXID      string `json:"txid"`
}

// BashynskyiBlock представляє блок
type BashynskyiBlock struct {
	Index        int                     `json:"index"`
	Timestamp    int64                   `json:"timestamp"`
	Transactions []BashynskyiTransaction `json:"transactions"`
	Proof        int                     `json:"proof"`
	PreviousHash string                  `json:"previous_hash"`
	Hash         string                  `json:"hash"`
}

// BashynskyiBlockchain представляє сам блокчейн
type BashynskyiBlockchain struct {
	Chain               []BashynskyiBlock
	CurrentTransactions []BashynskyiTransaction
}

// NewBashynskyiTransaction створює нову транзакцію та обчислює її TXID
func NewBashynskyiTransaction(sender, recipient string, amount int) BashynskyiTransaction {
	tx := BashynskyiTransaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	// Генерація TXID на основі даних транзакції
	data := sender + recipient + strconv.Itoa(amount)
	hash := sha256.Sum256([]byte(data))
	tx.TXID = hex.EncodeToString(hash[:])
	return tx
}

// CalculateHash обчислює хеш блоку на основі його полів
func (b *BashynskyiBlock) CalculateHash() string {
	// Перетворюємо транзакції в рядок (для простоти використаємо JSON)
	txBytes, _ := json.Marshal(b.Transactions)
	record := strconv.Itoa(b.Index) + strconv.FormatInt(b.Timestamp, 10) + string(txBytes) +
		strconv.Itoa(b.Proof) + b.PreviousHash
	h := sha256.Sum256([]byte(record))
	return hex.EncodeToString(h[:])
}

// NewBashynskyiBlockchain створює новий блокчейн з генезис-блоком
func NewBashynskyiBlockchain() *BashynskyiBlockchain {
	bc := &BashynskyiBlockchain{
		Chain:               []BashynskyiBlock{},
		CurrentTransactions: []BashynskyiTransaction{},
	}
	// Створюємо генезис-блок
	genesisBlock := bc.createGenesisBlock()
	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

// createGenesisBlock будує генезис-блок з попереднім хешем "Bashynskyi" та nonce = 19022006
// і підбирає timestamp так, щоб хеш блоку закінчувався на birthMonthSuffix ("02")
func (bc *BashynskyiBlockchain) createGenesisBlock() BashynskyiBlock {
	// Задані параметри
	index := 0
	previousHash := "Bashynskyi"
	proof := 19022006 // день місяць рік народження: 19 02 2006
	transactions := []BashynskyiTransaction{}

	// Початкова мітка часу (можна взяти поточну)
	timestamp := time.Now().Unix()
	// Шукаємо timestamp, при якому хеш закінчується на "02"
	for {
		block := BashynskyiBlock{
			Index:        index,
			Timestamp:    timestamp,
			Transactions: transactions,
			Proof:        proof,
			PreviousHash: previousHash,
		}
		block.Hash = block.CalculateHash()
		if strings.HasSuffix(block.Hash, birthMonthSuffix) {
			return block
		}
		timestamp++ // збільшуємо і пробуємо далі
	}
}

// AddBlock додає новий блок до ланцюга після майнінгу (пошуку proof, що дає потрібне закінчення хешу)
func (bc *BashynskyiBlockchain) AddBlock(transactions []BashynskyiTransaction) error {
	previousBlock := bc.Chain[len(bc.Chain)-1]
	newBlock := BashynskyiBlock{
		Index:        previousBlock.Index + 1,
		Timestamp:    time.Now().Unix(), // поточна мітка часу
		Transactions: transactions,
		Proof:        0, // почнемо з нуля
		PreviousHash: previousBlock.Hash,
	}

	// Майнінг: збільшуємо Proof, поки хеш не закінчиться на "02"
	for {
		newBlock.Hash = newBlock.CalculateHash()
		if strings.HasSuffix(newBlock.Hash, birthMonthSuffix) {
			break
		}
		newBlock.Proof++
	}

	bc.Chain = append(bc.Chain, newBlock)
	return nil
}

// AddTransaction додає транзакцію до поточного списку та повертає її TXID
func (bc *BashynskyiBlockchain) AddTransaction(transaction BashynskyiTransaction) string {
	bc.CurrentTransactions = append(bc.CurrentTransactions, transaction)
	return transaction.TXID
}

// String виводить блокчейн у гарному форматі
func (bc *BashynskyiBlockchain) String() string {
	var s string
	for i, block := range bc.Chain {
		s += fmt.Sprintf("Block %d:\n", i)
		s += fmt.Sprintf("  Index: %d\n", block.Index)
		s += fmt.Sprintf("  Timestamp: %d\n", block.Timestamp)
		s += fmt.Sprintf("  Transactions: %v\n", block.Transactions)
		s += fmt.Sprintf("  Proof: %d\n", block.Proof)
		s += fmt.Sprintf("  PreviousHash: %s\n", block.PreviousHash)
		s += fmt.Sprintf("  Hash: %s\n", block.Hash)
		s += fmt.Sprintf("  Hash ends with '%s': %v\n", birthMonthSuffix, strings.HasSuffix(block.Hash, birthMonthSuffix))
		s += "\n"
	}
	return s
}

func main() {
	// Ініціалізація блокчейну
	bc := NewBashynskyiBlockchain()
	fmt.Println("Genesis block created:")
	fmt.Println(bc)

	// Додаємо транзакцію
	tx1 := NewBashynskyiTransaction("Alice", "Bob", 10)
	txid := bc.AddTransaction(tx1)
	fmt.Printf("Transaction added with TXID: %s\n", txid)

	// Майнінг нового блоку з поточною транзакцією
	err := bc.AddBlock(bc.CurrentTransactions)
	if err != nil {
		fmt.Println("Error adding block:", err)
		return
	}
	// Очищаємо список поточних транзакцій після додавання блоку
	bc.CurrentTransactions = []BashynskyiTransaction{}

	fmt.Println("After mining a new block:")
	fmt.Println(bc)

	// Додамо ще одну транзакцію і ще один блок
	tx2 := NewBashynskyiTransaction("Bob", "Charlie", 5)
	bc.AddTransaction(tx2)
	bc.AddBlock(bc.CurrentTransactions)
	bc.CurrentTransactions = []BashynskyiTransaction{}

	fmt.Println("After second block:")
	fmt.Println(bc)

	// Перевірка, що всі хеші закінчуються на "02"
	fmt.Println("Verification that all block hashes end with '02':")
	for i, block := range bc.Chain {
		valid := strings.HasSuffix(block.Hash, birthMonthSuffix)
		fmt.Printf("Block %d: %v\n", i, valid)
	}
}
