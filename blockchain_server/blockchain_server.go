package main

import (
	"encoding/json"
	"goblockchain/block"
	"goblockchain/utils"
	"goblockchain/wallet"
	"io"
	"log"
	"net/http"
	"strconv"
)

var cache map[string]*block.Blockchain = make(map[string]*block.Blockchain)

type BlockchainServer struct {
	port uint16
}

func NewBlockchainServer(port uint16) *BlockchainServer {
	return &BlockchainServer{port}
}

func (bcs *BlockchainServer) Port() uint16 {
	return bcs.port
}

func (bcs *BlockchainServer) GetBlockchain() *block.Blockchain {
	bc, ok := cache["blockchain"]
	if !ok {
		minersWallet := wallet.NewWallet()
		bc = block.NewBlockchain(minersWallet.BlockchainAddress(), bcs.Port())
		cache["blockchain"] = bc
		log.Printf("private_key %v\n", minersWallet.PrivateKeyStr())
		log.Printf("public_key %v\n", minersWallet.PublicKeyStr())
		log.Printf("blockchain_address %v\n", minersWallet.BlockchainAddress())
	}

	return bc
}

func (bcs *BlockchainServer) GetChain(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		m, _ := bc.MarshalJSON()
		io.WriteString(w, string(m[:]))
		break
	default:
		log.Println("ERROR! Invalid HTTP Method!")
	}
}

func (bcs *BlockchainServer) Transactions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		transactions := bc.TransactionPool()
		m, _ := json.Marshal(struct {
			Transactions []*block.Transaction `json:"transactions"`
			Length       int                  `json:"length"`
		}{
			Transactions: transactions,
			Length:       len(transactions),
		})

		io.WriteString(w, string(m))
		break
	case http.MethodPost:
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("ERROR: %s", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)

		/*
			sender string,
				recipient string,
				value float32,
				senderPublicKey *ecdsa.PublicKey,
				signature *utils.Signature
		*/
		bc := bcs.GetBlockchain()
		isCreated := bc.CreateTransaction(
			*t.SenderBlockchainAddress,
			*t.RecipientBlockchainAddress,
			*t.Value,
			publicKey,
			signature,
		)

		w.Header().Add("Content-Type", "application/json")
		var m []byte

		if !isCreated {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("fail")
		} else {
			w.WriteHeader(http.StatusCreated)
			m = utils.JsonStatus("success")
		}

		io.WriteString(w, string(m))

		break

	case http.MethodPut:
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("ERROR: %s", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)

		/*
			sender string,
				recipient string,
				value float32,
				senderPublicKey *ecdsa.PublicKey,
				signature *utils.Signature
		*/
		bc := bcs.GetBlockchain()
		isUpdated := bc.AddTransaction(
			*t.SenderBlockchainAddress,
			*t.RecipientBlockchainAddress,
			*t.Value,
			publicKey,
			signature,
		)

		w.Header().Add("Content-Type", "application/json")
		var m []byte

		if !isUpdated {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("fail")
		} else {
			m = utils.JsonStatus("success")
		}

		io.WriteString(w, string(m))

		break
	case http.MethodDelete:
		bc := bcs.GetBlockchain()
		bc.ClearTransactionPool()
		io.WriteString(w, string(utils.JsonStatus("success")))
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR! Invalid HTTP Method!")
	}
}

func (bcs *BlockchainServer) Mine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockchain()
		isMined := bc.Mining()
		var m []byte
		if !isMined {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("fail")
		} else {
			m = utils.JsonStatus("success")
		}

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
		return
	case http.MethodPost:
		return
	default:
		log.Println("Invalid HTTP method!")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StartMine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockchain()
		bc.StartMining()

		m := utils.JsonStatus("success")
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
		return
	case http.MethodPost:
		return
	default:
		log.Println("Invalid HTTP method!")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Amount(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		blockchainAddress := req.URL.Query().Get("blockchain_address")
		amount := bcs.GetBlockchain().CalculateTotalAmount(blockchainAddress)

		ar := &block.AmountResponse{Amount: amount}
		m, _ := ar.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Println("ERROR! Invalid Http Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Consensus(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPut:
		bc := bcs.GetBlockchain()
		replaced := bc.ResolveConflicts()

		w.Header().Add("Content-Type", "application/json")
		if replaced {
			io.WriteString(w, string(utils.JsonStatus("success")))
		} else {
			io.WriteString(w, string(utils.JsonStatus("fail")))
		}
	default:
		log.Println("Invalid HTTP method!")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Run() {
	bcs.GetBlockchain().Run()

	http.HandleFunc("/chain", bcs.GetChain)
	http.HandleFunc("/transactions", bcs.Transactions)
	http.HandleFunc("/mine", bcs.Mine)
	http.HandleFunc("/mine/start", bcs.StartMine)
	http.HandleFunc("/amount", bcs.Amount)
	http.HandleFunc("/consensus", bcs.Consensus)

	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(int(bcs.Port())), nil))
}
