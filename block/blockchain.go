package block

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"goblockchain/utils"
	"log"
	"strings"
	"time"
)

const (
	MINING_DIFICULTY = 3
	MINING_SENDER    = "THE BLOCKCHAIN"
	MINING_REWARD    = 1.0
)

type Block struct {
	timestamps   int64
	nonce        int
	previousHash [32]byte
	transactions []*Transaction
}

type Blockchain struct {
	transactionPool   []*Transaction
	chain             []*Block
	blockchainAddress string
}

type Transaction struct {
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	value                      float32
}

func (b *Block) Print() {
	fmt.Printf("timestamps      %d\n", b.timestamps)
	fmt.Printf("nonce           %d\n", b.nonce)
	fmt.Printf("previousHash    %x\n", b.previousHash)

	for _, t := range b.transactions {
		t.Print()
	}
}

func (b *Block) Hash() [32]byte {
	m, _ := json.Marshal(b)
	return sha256.Sum256([]byte(m))
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp    int64          `json:"timestamp"`
		Nonce        int            `json:"nonce"`
		PreviousHash [32]byte       `json:"previous_hash"`
		Transactions []*Transaction `json:"transactions"`
	}{
		Timestamp:    b.timestamps,
		Nonce:        b.nonce,
		PreviousHash: b.previousHash,
		Transactions: b.transactions,
	})
}

func NewBlock(nonce int, previousHash [32]byte, transactions []*Transaction) *Block {

	block := new(Block)
	block.timestamps = time.Now().UnixNano()
	block.nonce = nonce
	block.previousHash = previousHash
	block.transactions = transactions
	return block
}

func NewBlockchain(blockchainAddress string) *Blockchain {
	b := &Block{}
	bc := new(Blockchain)
	bc.blockchainAddress = blockchainAddress
	bc.CreateBlock(0, b.Hash())
	return bc
}

func (bc *Blockchain) LastBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash [32]byte) *Block {
	b := NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{}
	return b
}

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		fmt.Printf("%s Chain %d %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 25))
}

func (bc *Blockchain) AddTransaction(sender string,
	recipient string,
	value float32,
	senderPublicKey *ecdsa.PublicKey,
	signature *utils.Signature) bool {

	t := NewTransaction(sender, recipient, value)

	if sender == MINING_SENDER {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	if bc.VerifyTransactionSignature(senderPublicKey, signature, t) {

		/*
			if bc.CalculateTotalAmount(sender) < value {
				log.Println("ERROR: Not enough balance in wallet")
				return false
			}
		*/

		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	log.Println("ERROR: Verify Transaction!")
	return false
}

func (bc *Blockchain) VerifyTransactionSignature(senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool {
	m, _ := json.Marshal(t)
	h := sha256.Sum256([]byte(m))
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
}

func (bc *Blockchain) CopyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _, t := range bc.transactionPool {
		transactions = append(transactions,
			NewTransaction(
				t.senderBlockchainAddress,
				t.recipientBlockchainAddress,
				t.value))
	}

	return transactions
}

func (bc *Blockchain) ValidProof(nonce int, previousHash [32]byte, transactions []*Transaction, difficulty int) bool {
	zeros := strings.Repeat("0", MINING_DIFICULTY)
	guessBlock := Block{0, nonce, previousHash, transactions}

	guessHashStr := fmt.Sprintf("%x", guessBlock.Hash())

	if guessHashStr[:difficulty] == zeros {
		fmt.Println(strings.Repeat("=", 40))
		fmt.Println(guessHashStr)
		fmt.Println(strings.Repeat("=", 40))
	}
	return guessHashStr[:difficulty] == zeros
}

func (bc *Blockchain) ProofOfWork() int {
	transactions := bc.CopyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	nonce := 0

	for !bc.ValidProof(nonce, previousHash, transactions, MINING_DIFICULTY) {
		nonce += 1
	}

	return nonce
}

func (bc *Blockchain) Mining() bool {
	bc.AddTransaction(MINING_SENDER, bc.blockchainAddress, MINING_REWARD, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	bc.CreateBlock(nonce, previousHash)
	log.Println("actions=mining, status=success")
	return true
}

func (bc *Blockchain) CalculateTotalAmount(blockchainAddress string) float32 {
	var totalAmount float32 = 0
	for _, b := range bc.chain {
		for _, t := range b.transactions {
			if t.recipientBlockchainAddress == blockchainAddress {
				totalAmount += t.value
			}

			if t.senderBlockchainAddress == blockchainAddress {
				totalAmount -= t.value
			}
		}
	}

	return totalAmount
}

func NewTransaction(sender string, recipient string, value float32) *Transaction {
	return &Transaction{sender, recipient, value}
}

func (t *Transaction) Print() {
	fmt.Printf("%s\n", strings.Repeat("-", 40))
	fmt.Printf(" sender_blockchain_address       %s\n", t.senderBlockchainAddress)
	fmt.Printf(" recipient_blockchain_address    %s\n", t.recipientBlockchainAddress)
	fmt.Printf(" value                           %.1f\n", t.value)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender    string  `json:"sender_blockchain_address"`
		Recipient string  `json:"recipient_blockchain_address"`
		Value     float32 `json:"value"`
	}{
		Sender:    t.senderBlockchainAddress,
		Recipient: t.recipientBlockchainAddress,
		Value:     t.value,
	})
}
