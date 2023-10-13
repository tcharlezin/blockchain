package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goblockchain/block"
	"goblockchain/utils"
	"goblockchain/wallet"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
)

type WalletServer struct {
	port    uint16
	gateway string
}

const tempDir = "wallet_server/templates"

func NewWalletServer(port uint16, gateway string) *WalletServer {
	return &WalletServer{
		port:    port,
		gateway: gateway,
	}
}

func (ws *WalletServer) Port() uint16 {
	return ws.port
}

func (ws *WalletServer) Gateway() string {
	return ws.gateway
}

func (ws *WalletServer) Index(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		t, _ := template.ParseFiles(path.Join(tempDir, "index.html"))
		t.Execute(w, "")
	default:
		log.Println("ERROR! Invalid HTTP Method!")
	}
}

func (ws *WalletServer) Wallet(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "Application/Json")
		mywallet := wallet.NewWallet()
		m, _ := mywallet.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR! Invalid HTTP Method!")
	}
}

func (ws *WalletServer) CreateTransaction(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		decoder := json.NewDecoder(req.Body)
		var transactionRequest wallet.TransactionRequest
		err := decoder.Decode(&transactionRequest)

		if err != nil {
			log.Printf("ERROR: %v", err)
			fmt.Println(fmt.Sprintf("1111 %s", err))
			io.WriteString(w, string(utils.JsonStatus("Failed!")))
			return
		}

		if !transactionRequest.Validate() {
			log.Printf("ERROR: Missing fields!")
			io.WriteString(w, string(utils.JsonStatus("Invalid!")))
			return
		}

		publicKey := utils.PublicKeyFromString(*transactionRequest.SenderPublicKey)
		privateKey := utils.PrivateKeyFromString(*transactionRequest.SenderPrivateKey, publicKey)

		value, err := strconv.ParseFloat(*transactionRequest.Value, 32)

		if err != nil {
			log.Println("ERROR! Parse error!")
			io.WriteString(w, string(utils.JsonStatus("Failed!")))
		}

		value32 := float32(value)

		transaction := wallet.NewTransaction(
			privateKey,
			publicKey,
			*transactionRequest.SenderBlockchainAddress,
			*transactionRequest.RecipientBlockchainAddress,
			value32)

		signature := transaction.GenerateSignature()
		signatureStr := signature.String()

		bt := &block.TransactionRequest{
			SenderBlockchainAddress:    transactionRequest.SenderBlockchainAddress,
			RecipientBlockchainAddress: transactionRequest.RecipientBlockchainAddress,
			SenderPublicKey:            transactionRequest.SenderPublicKey,
			Value:                      &value32,
			Signature:                  &signatureStr,
		}

		m, _ := json.Marshal(bt)
		buf := bytes.NewBuffer(m)

		resp, err := http.Post(ws.Gateway()+"/transactions", "application/json", buf)

		if err != nil {
			fmt.Println(fmt.Sprintf("%s", err))
			io.WriteString(w, fmt.Sprintf("%s", err))
			return
		}

		fmt.Println(fmt.Sprintf("%s", resp.Body))

		if resp.StatusCode == 201 {
			io.WriteString(w, string(utils.JsonStatus("Success")))
			return
		}

		io.WriteString(w, string(utils.JsonStatus("Fail")))
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR! Invalid HTTP Method!")
	}
}

func (ws *WalletServer) WalletAmount(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		blockchainAddress := req.URL.Query().Get("blockchain_address")
		endpoint := fmt.Sprintf("%s/amount", ws.Gateway())

		client := &http.Client{}
		bcsReq, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		q := bcsReq.URL.Query()
		q.Add("blockchain_address", blockchainAddress)
		bcsReq.URL.RawQuery = q.Encode()

		bcsResp, err := client.Do(bcsReq)
		if err != nil {
			log.Printf("ERROR: %v", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		w.Header().Add("Content-Type", "application/json")
		if bcsResp.StatusCode == 200 {
			decoder := json.NewDecoder(bcsResp.Body)
			var bar block.AmountResponse
			err := decoder.Decode(&bar)

			if err != nil {
				log.Printf("ERROR: %v", err)
				io.WriteString(w, string(utils.JsonStatus("fail")))
				return
			}

			m, _ := json.Marshal(struct {
				Message string  `json:"message"`
				Amount  float32 `json:"amount"`
			}{
				Message: "success",
				Amount:  bar.Amount,
			})

			io.WriteString(w, string(m[:]))
		} else {
			io.WriteString(w, string(utils.JsonStatus("fail")))
		}
	default:
		log.Printf("ERROR: Invalid HTTP Method!")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (ws *WalletServer) Run() {
	http.HandleFunc("/", ws.Index)
	http.HandleFunc("/wallet", ws.Wallet)
	http.HandleFunc("/wallet/amount", ws.WalletAmount)
	http.HandleFunc("/transaction", ws.CreateTransaction)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(int(ws.Port())), nil))
}
