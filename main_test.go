package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	mockhttp "github.com/karupanerura/go-mock-http-response"
	"testing"
)

func mockResponse(statusCode int, headers map[string]string, body []byte) {
	http.DefaultClient = mockhttp.NewResponseMock(statusCode, headers, body).MakeClient()
}

func TestTxnHandler(t *testing.T) {
	//t.Parallel()
	testCases := []struct {
		name        string
		method      string
		webhookbody string
		message     string
		mockresp    []byte
	}{
		{"empty GET", http.MethodGet, "", "INFO: empty body", []byte{}},
		{"empty POST", http.MethodPost, "", "INFO: empty body", []byte{}},
		{
			"invalid json",
			http.MethodPost,
			`{"foo":"bar}`,
			"ERROR: failed to unmarshal web hook payload",
			[]byte{},
		},
		{
			"transaction above threshold",
			http.MethodPost,
			`{"data":{"id": "tx_1234_above", "amount": 2500}}`,
			"INFO: transfer successful",
			[]byte(`{"transaction":{"amount":2500, "account_balance":2574, "merchant": null}}`),
		},
		{
			"transaction below threshold",
			http.MethodPost,
			`{"data":{"id": "tx_4567_below", "amount": 500}}`,
			"INFO: ignoring inbound transaction below sweep threshold",
			[]byte(`{"transaction":{"amount": 500, "account_balance": 75412, "merchant": null}}`),
		},
		{
			"transaction above threshold with 0 balance",
			http.MethodPost,
			`{"data":{"id": "tx_1234_zero", "amount": 2500}}`,
			"INFO: doing nothing as balance <= 0",
			[]byte(`{"transaction":{"amount":2500, "account_balance":2500, "merchant": null}}`),
		},
		{
			"transaction above threshold with negative balance",
			http.MethodPost,
			`{"data":{"id": "tx_1234_negative", "amount": 2500}}`,
			"INFO: doing nothing as balance <= 0",
			[]byte(`{"transaction":{"amount":2500, "account_balance":2474, "merchant": null}}`),
		},
		{
			"duplicate transaction above threshold 1",
			http.MethodPost,
			`{"data":{"id": "tx_1234_dup", "amount": 2500}}`,
			"INFO: transfer successful",
			[]byte(`{"transaction":{"amount":2500, "account_balance":2574, "merchant": null}}`),
		},
		{
			"duplicate transaction above threshold 2",
			http.MethodPost,
			`{"data":{"id": "tx_1234_dup", "amount": 2500}}`,
			"INFO: ignoring duplicate webhook delivery",
			[]byte{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			s.SweepThreshold = 1000
			// Set a mock response, if needed.
			if len(tc.mockresp) > 0 {
				mockResponse(http.StatusOK, map[string]string{"Content-Type": "application/json"}, tc.mockresp)
			}
			// Use a faux logger so we can parse the content to find our debug messages to confirm our tests
			var fauxLog bytes.Buffer
			log.SetOutput(&fauxLog)
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.webhookbody))
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(TxnHandler)
			handler.ServeHTTP(rr, req)
			if !strings.Contains(fauxLog.String(), tc.message) {
				t.Errorf("'%v' failed.\nGot:\n%v\nExpected:\n%v", tc.name, fauxLog.String(), tc.message)
			}
		})
	}
}
