package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	monzo "github.com/tjvr/go-monzo"
)

// Settings defines the structure of all the configuration options pulled from environment variables.
type Settings struct {
	Port                string  `required:"true" envconfig:"PORT"` // Default: 5000 when run locally
	PersonalAccessToken string  `required:"true" split_words:"true"`
	SweepPotId          string  `required:"true" split_words:"true"`
	SweepThreshold      float64 `required:"true" split_words:"true"`
	AccountId           string  `required:"true" split_words:"true"`
}

// WebHookPayload defines the structure of the Monzo webhook payload, but only for the fields we're interested in.
type WebHookPayload struct {
	Type string      `json:"type"`
	Data WebHookData `json:"data"`
}

// WebHookData defines the structure of the Monzo webhook data attribute, but only for the fields we're interested in.
type WebHookData struct {
	AccountId     string    `json:"account_id"`
	Amount        float64   `json:"amount"`
	Create        time.Time `json:"created"`
	TransactionId string    `json:"id"`
	IsLoad        bool      `json:"is_load"`
}

var s Settings

func main() {
	log.SetFlags(0)
	err := envconfig.Process("monzo", &s)
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc("/", TxnHandler)
	http.ListenAndServe(":"+s.Port, nil)
}

func TxnHandler(w http.ResponseWriter, r *http.Request) {
	// Return OK as soon as we've received the payload - the webhook doesn't care what we do with the payload so no point holding things back.
	w.WriteHeader(http.StatusOK)

	// Grab body early as we'll need it later
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) == "" {
		log.Println("INFO: empty body, pretending all is OK")
		return
	}

	wh := new(WebHookPayload)
	err := json.Unmarshal([]byte(body), &wh)
	if err != nil {
		log.Println("ERROR: failed to unmarshal web hook payload:", err)
		return
	}

	// Store the webhook uid in an environment variable and use to try catch duplicate deliveries
	ltu, _ := os.LookupEnv("LAST_TRANSACTION_ID")
	if ltu != "" && ltu == wh.Data.TransactionId {
		log.Println("INFO: ignoring duplicate webhook delivery")
		return
	}

	os.Setenv("LAST_TRANSACTION_ID", wh.Data.TransactionId)

	if s.SweepThreshold <= 0.0 || wh.Data.Amount < s.SweepThreshold {
		log.Println("INFO: ignoring inbound transaction below sweep threshold")
		return
	}

	// We've got this far so we can assume the amount is greater than the threshold
	cl := monzo.Client{
		BaseURL:     "https://api.monzo.com",
		AccessToken: s.PersonalAccessToken,
	}

	log.Printf("INFO: threshold: %v\n", s.SweepThreshold)
	txn, err := cl.Transaction(wh.Data.TransactionId)
	if err != nil {
		log.Println("ERROR: problem getting transaction ", wh.Data.TransactionId)
		log.Println(err.Error())
	}
	bal := (txn.AccountBalance - txn.Amount)
	log.Printf("INFO: balance before: %v", bal)

	if bal <= 0 {
		log.Println("INFO: doing nothing as balance <= 0")
	}

	resp, err := cl.Deposit(&monzo.DepositRequest{
		PotID:          s.SweepPotId,
		AccountID:      s.AccountId,
		Amount:         bal,
		IdempotencyKey: wh.Data.TransactionId,
	})

	if err != nil {
		log.Printf("ERROR: problem transferring to pot '%v'", s.SweepPotId)
		log.Println(err.Error())
	}
	log.Printf("INFO: transfer successful (New bal: %.2f | %.2f)", float64(resp.Balance/100), float64(bal/100))
}
