package block

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"goblockchain/utils"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	MINING_DIFICULTY                  = 3
	MINING_SENDER                     = "THE BLOCKCHAIN"
	MINING_REWARD                     = 1.0
	MINING_TIME_SEC                   = 20
	BLOCKCHAIN_PORT_RANGE_START       = 8000
	BLOCKCHAIN_PORT_RANGE_END         = 8003
	NEIGHBOR_IP_RANGE_START           = 0
	NEIGHBOR_IP_RANGE_END             = 1
	BLOCKCHAIN_NEIGHBOR_SYNC_TIME_SEC = 5
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
	port              uint16
	mux               sync.Mutex

	neighbors    []string
	muxNeighbors sync.Mutex
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
		PreviousHash string         `json:"previous_hash"`
		Transactions []*Transaction `json:"transactions"`
	}{
		Timestamp:    b.timestamps,
		Nonce:        b.nonce,
		PreviousHash: fmt.Sprintf("%x", b.previousHash),
		Transactions: b.transactions,
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {

	var previousHash string

	v := &struct {
		Timestamp    *int64          `json:"timestamp"`
		Nonce        *int            `json:"nonce"`
		PreviousHash *string         `json:"previous_hash"`
		Transactions *[]*Transaction `json:"transactions"`
	}{
		Timestamp:    &b.timestamps,
		Nonce:        &b.nonce,
		PreviousHash: &previousHash,
		Transactions: &b.transactions,
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	ph, _ := hex.DecodeString(*v.PreviousHash)
	copy(b.previousHash[:], ph[:32])
	return nil
}

func NewBlock(nonce int, previousHash [32]byte, transactions []*Transaction) *Block {

	block := new(Block)
	block.timestamps = time.Now().UnixNano()
	block.nonce = nonce
	block.previousHash = previousHash
	block.transactions = transactions
	return block
}

func (b *Block) PreviousHash() [32]byte {
	return b.previousHash
}

func (b *Block) Nonce() int {
	return b.nonce
}

func (b *Block) Transactions() []*Transaction {
	return b.transactions
}

func NewBlockchain(blockchainAddress string, port uint16) *Blockchain {
	b := &Block{}
	bc := new(Blockchain)
	bc.blockchainAddress = blockchainAddress
	bc.CreateBlock(0, b.Hash())
	bc.port = port
	return bc
}

func (bc *Blockchain) Chain() []*Block {
	return bc.chain
}

func (bc *Blockchain) Run() {
	bc.StartSyncNeighbors()
	bc.ResolveConflicts()
	bc.StartMining()
}

func (bc *Blockchain) SetNeighbors() {
	bc.neighbors = utils.FindNeighbors(
		utils.GetHost(),
		bc.port,
		NEIGHBOR_IP_RANGE_START,
		NEIGHBOR_IP_RANGE_END,
		BLOCKCHAIN_PORT_RANGE_START,
		BLOCKCHAIN_PORT_RANGE_END,
	)

	log.Printf("%v", bc.neighbors)
}

func (bc *Blockchain) SyncNeighbors() {
	bc.muxNeighbors.Lock()
	defer bc.muxNeighbors.Unlock()

	bc.SetNeighbors()
}

func (bc *Blockchain) StartSyncNeighbors() {
	bc.SyncNeighbors()
	_ = time.AfterFunc(time.Second*BLOCKCHAIN_NEIGHBOR_SYNC_TIME_SEC, bc.StartSyncNeighbors)
}

func (bc *Blockchain) TransactionPool() []*Transaction {
	return bc.transactionPool
}

func (bc *Blockchain) ClearTransactionPool() {
	bc.transactionPool = bc.transactionPool[:0]
}

func (bc *Blockchain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*Block `json:"chain"`
	}{
		Blocks: bc.chain,
	})
}

func (bc *Blockchain) LastBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) UnmarshalJSON(data []byte) error {
	v := &struct {
		Blocks *[]*Block `json:"chain"`
	}{
		Blocks: &bc.chain,
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash [32]byte) *Block {
	b := NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{}

	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/transactions", n)
		client := &http.Client{}
		req, _ := http.NewRequest("DELETE", endpoint, nil)
		resp, _ := client.Do(req)
		log.Printf("%v", resp)
	}

	return b
}

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		fmt.Printf("%s Chain %d %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 25))
}

func (bc *Blockchain) CreateTransaction(sender string,
	recipient string,
	value float32,
	senderPublicKey *ecdsa.PublicKey,
	signature *utils.Signature) bool {
	isTransacted := bc.AddTransaction(sender, recipient, value, senderPublicKey, signature)

	if isTransacted {
		for _, n := range bc.neighbors {
			publicKeyStr := fmt.Sprintf("%064x%064x", senderPublicKey.X.Bytes(), senderPublicKey.Y.Bytes())
			signatureStr := signature.String()
			bt := &TransactionRequest{
				SenderBlockchainAddress:    &sender,
				RecipientBlockchainAddress: &recipient,
				SenderPublicKey:            &publicKeyStr,
				Value:                      &value,
				Signature:                  &signatureStr,
			}

			m, _ := json.Marshal(bt)
			buf := bytes.NewBuffer(m)
			endpoint := fmt.Sprintf("http://%s/transactions", n)
			client := &http.Client{}
			req, _ := http.NewRequest("PUT", endpoint, buf)
			resp, _ := client.Do(req)
			log.Printf("%v", resp)
		}
	}

	return isTransacted
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

		if bc.CalculateTotalAmount(sender) < value {
			log.Println("ERROR: Not enough balance in wallet")
			return false
		}

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
	bc.mux.Lock()
	defer bc.mux.Unlock()

	/*
		if len(bc.transactionPool) == 0 {
			return false
		}
	*/

	bc.AddTransaction(MINING_SENDER, bc.blockchainAddress, MINING_REWARD, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	bc.CreateBlock(nonce, previousHash)
	log.Println("actions=mining, status=success")

	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/consensus", n)
		client := &http.Client{}
		req, _ := http.NewRequest("PUT", endpoint, nil)
		resp, _ := client.Do(req)
		log.Printf("%v", resp)
	}

	return true
}

func (bc *Blockchain) StartMining() {
	bc.Mining()
	_ = time.AfterFunc(time.Second*MINING_TIME_SEC, bc.StartMining)
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

func (bc *Blockchain) ValidChain(chain []*Block) bool {
	preBlock := chain[0]
	currentIndex := 1

	for currentIndex < len(chain) {
		b := chain[currentIndex]
		if b.previousHash != preBlock.Hash() {
			return false
		}

		if !bc.ValidProof(b.Nonce(), b.PreviousHash(), b.Transactions(), MINING_DIFICULTY) {
			return false
		}

		preBlock = b
		currentIndex += 1
	}

	return true
}

func (bc *Blockchain) ResolveConflicts() bool {
	var longestChain []*Block = nil
	maxLength := len(bc.chain)

	for _, n := range bc.neighbors {

		endpoint := fmt.Sprintf("http://%s/chain", n)
		resp, _ := http.Get(endpoint)
		if resp.StatusCode == 200 {
			var bcResp Blockchain
			decoder := json.NewDecoder(resp.Body)
			_ = decoder.Decode(&bcResp)

			chain := bcResp.Chain()
			if len(chain) > maxLength && bc.ValidChain(chain) {
				maxLength = len(chain)
				longestChain = chain
			}
		}
	}

	if longestChain != nil {
		bc.chain = longestChain
		log.Printf("Resolve conflicts replaced")
		return true
	}

	log.Printf("Resolve conflicts not replaced")
	return false
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

func (t *Transaction) UnmarshalJSON(data []byte) error {
	v := &struct {
		Sender    *string  `json:"sender_blockchain_address"`
		Recipient *string  `json:"recipient_blockchain_address"`
		Value     *float32 `json:"value"`
	}{
		Sender:    &t.senderBlockchainAddress,
		Recipient: &t.recipientBlockchainAddress,
		Value:     &t.value,
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	return nil
}

type TransactionRequest struct {
	SenderBlockchainAddress    *string  `json:"sender_blockchain_address"`
	RecipientBlockchainAddress *string  `json:"recipient_blockchain_address"`
	SenderPublicKey            *string  `json:"sender_public_key"`
	Value                      *float32 `json:"value"`
	Signature                  *string  `json:"signature"`
}

func (tr *TransactionRequest) Validate() bool {
	if tr.SenderBlockchainAddress == nil ||
		tr.RecipientBlockchainAddress == nil ||
		tr.SenderPublicKey == nil ||
		tr.Value == nil ||
		tr.Signature == nil {
		return false
	}

	return true
}

type AmountResponse struct {
	Amount float32 `json:"amount"`
}

func (ar *AmountResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount float32 `json:"amount"`
	}{
		Amount: ar.Amount,
	})
}
