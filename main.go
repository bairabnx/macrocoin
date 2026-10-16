package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Data Structures
type Transaction struct {
	Sender    string  `json:"sender"`
	Recipient string  `json:"recipient"`
	Amount    float64 `json:"amount"`
}

type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	Nonce        int           `json:"nonce"`
	PreviousHash string        `json:"previous_hash"`
	Hash         string        `json:"hash"`
}

type Blockchain struct {
	Chain        []Block       `json:"chain"`
	Transactions []Transaction `json:"transactions"`
}

// Global Ledger State
var nodeBlockchain = Blockchain{
	Chain:        []Block{},
	Transactions: []Transaction{},
}

// Proof of Work Mechanics
func (b *Blockchain) ProofOfWork(lastNonce int) int {
	nonce := 0
	for !b.ValidProof(lastNonce, nonce) {
		nonce++
	}
	return nonce
}

func (b *Blockchain) ValidProof(lastNonce, nonce int) bool {
	guess := strconv.Itoa(lastNonce) + strconv.Itoa(nonce)
	guessHash := fmt.Sprintf("%x", sha256.Sum256([]byte(guess)))
	return strings.HasPrefix(guessHash, "0000")
}

func main() {
	port := flag.String("port", "8080", "The port to listen on")
	flag.Parse()

	// Route 1: View Ledger
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodeBlockchain.Chain)
	})

	// Route 2: New Transaction
	http.HandleFunc("/transactions/new", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		sender := r.URL.Query().Get("sender")
		recipient := r.URL.Query().Get("recipient")
		amountStr := r.URL.Query().Get("amount")

		if sender == "" || recipient == "" || amountStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing parameters"})
			return
		}

		amount, _ := strconv.ParseFloat(amountStr, 64)

		// Calculate Sender Balance
		senderBalance := 0.0
		for _, block := range nodeBlockchain.Chain {
			for _, tx := range block.Transactions {
				if tx.Sender == sender {
					senderBalance -= tx.Amount
				}
				if tx.Recipient == sender {
					senderBalance += tx.Amount
				}
			}
		}

		if senderBalance < amount && sender != "0" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Transaction rejected: Insufficient funds."})
			return
		}

		tx := Transaction{Sender: sender, Recipient: recipient, Amount: amount}
		nodeBlockchain.Transactions = append(nodeBlockchain.Transactions, tx)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Success! Transaction staged in mempool."})
	})

	// Route 3: Mine Block
	http.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		var lastBlock Block
		var lastNonce int
		var prevHash string

		if len(nodeBlockchain.Chain) > 0 {
			lastBlock = nodeBlockchain.Chain[len(nodeBlockchain.Chain)-1]
			lastNonce = lastBlock.Nonce
			prevHash = lastBlock.Hash
		} else {
			lastNonce = 100
			prevHash = "1"
		}

		// Reward transaction (50 coins to miner)
		rewardTx := Transaction{
			Sender:    "0",
			Recipient: "3059301306072a8648ce3d020106082a8648ce3d03010703420004604bbd70065b2b686918704ecb47da16ea77c7f250e5224127da2fc46d1360c6f61ef314e11a9f12760957c0e244bcc398f6286c02d9b6862b2d44df8e747816"
			Amount:    50.0,
		}
		nodeBlockchain.Transactions = append(nodeBlockchain.Transactions, rewardTx)

		nonce := nodeBlockchain.ProofOfWork(lastNonce)

		newBlock := Block{
			Index:        len(nodeBlockchain.Chain) + 1,
			Timestamp:    time.Now().Unix(),
			Transactions: nodeBlockchain.Transactions,
			Nonce:        nonce,
			PreviousHash: prevHash,
		}

		blockBytes, _ := json.Marshal(newBlock)
		newBlock.Hash = fmt.Sprintf("%x", sha256.Sum256(blockBytes))

		nodeBlockchain.Chain = append(nodeBlockchain.Chain, newBlock)
		nodeBlockchain.Transactions = []Transaction{}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Congratulations! A new block has been successfully mined!"})
	})

	// Route 4: Balance Check
	http.HandleFunc("/balance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		address := r.URL.Query().Get("address")
		if address == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing address"})
			return
		}

		balance := 0.0
		for _, block := range nodeBlockchain.Chain {
			for _, tx := range block.Transactions {
				if tx.Sender == address {
					balance -= tx.Amount
				}
				if tx.Recipient == address {
					balance += tx.Amount
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"address": address, "balance": balance})
	})

	// Route 5: Consensus Node Synchronization
	http.HandleFunc("/nodes/resolve", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		peerURL := "http://localhost:8081/" // Example secondary local node peer
		resp, err := http.Get(peerURL)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "Could not connect to peer node."})
			return
		}
		defer resp.Body.Close()

		var peerChain []Block
		if err := json.NewDecoder(resp.Body).Decode(&peerChain); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read data package from peer."})
			return
		}

		if len(peerChain) > len(nodeBlockchain.Chain) {
			nodeBlockchain.Chain = peerChain
			json.NewEncoder(w).Encode(map[string]string{"message": "Consensus reached! Local chain replaced by peer network history.", "status": "updated"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Consensus complete. Local blockchain is already authoritative.", "status": "synchronized"})
	})

	fmt.Printf("Macrocoin Network Server booted successfully on http://0.0.0:%s\n", *port)
	http.ListenAndServe("0.0.0.0:"+*port, nil)
}

