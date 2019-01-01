package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mockhttp "github.com/karupanerura/go-mock-http-response"
)

func mockResponse(statusCode int, headers map[string]string, body []byte) {
	http.DefaultClient = mockhttp.NewResponseMock(statusCode, headers, body).MakeClient()
}

func TestTxnHandler(t *testing.T) {
	//t.Parallel()
	testCases := []struct {
		name             string
		method           string
		webhookPayload   string
		status           int
		logMessage       string
		mockResponseBody []byte
	}{
		{
			name:             "empty GET",
			method:           http.MethodGet,
			status:           http.StatusOK,
			webhookPayload:   "",
			logMessage:       "INFO: empty body",
			mockResponseBody: []byte{},
		},
		{
			name:             "empty POST",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   "",
			logMessage:       "INFO: empty body",
			mockResponseBody: []byte{},
		},
		{
			name:             "invalid json",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"foo":"bar}`,
			logMessage:       "ERROR: failed to unmarshal web hook payload",
			mockResponseBody: []byte{},
		},
		{
			name:             "transaction above threshold",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_1234_above", "amount": 2500}}`,
			logMessage:       "INFO: transfer successful",
			mockResponseBody: []byte(`{"transaction":{"amount":2500, "account_balance":2574, "merchant": null}}`),
		},
		{
			name:             "transaction below threshold",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_4567_below", "amount": 500}}`,
			logMessage:       "INFO: ignoring inbound transaction (500) below sweep threshold (1000)",
			mockResponseBody: []byte(`{"transaction":{"amount": 500, "account_balance": 75412, "merchant": null}}`),
		},
		{
			name:             "outbound transaction",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_4567_outwards", "amount": -500}}`,
			logMessage:       "INFO: ignoring inbound transaction (-500) below sweep threshold (1000)",
			mockResponseBody: []byte(`{"transaction":{"amount": 500, "account_balance": 75412, "merchant": null}}`),
		},
		{
			name:             "transaction above threshold with 0 balance",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_1234_zero", "amount": 2500}}`,
			logMessage:       "INFO: doing nothing as balance <= 0",
			mockResponseBody: []byte(`{"transaction":{"amount":2500, "account_balance":2500, "merchant": null}}`),
		},
		{
			name:             "transaction above threshold with negative balance",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_1234_negative", "amount": 2500}}`,
			logMessage:       "INFO: doing nothing as balance <= 0",
			mockResponseBody: []byte(`{"transaction":{"amount":2500, "account_balance":2474, "merchant": null}}`),
		},
		{
			name:             "duplicate transaction above threshold 1",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_1234_dup", "amount": 2500}}`,
			logMessage:       "INFO: transfer successful",
			mockResponseBody: []byte(`{"transaction":{"amount":2500, "account_balance":2574, "merchant": null}}`),
		},
		{
			name:             "duplicate transaction above threshold 2",
			method:           http.MethodPost,
			status:           http.StatusOK,
			webhookPayload:   `{"data":{"id": "tx_1234_dup", "amount": 2500}}`,
			logMessage:       "INFO: ignoring duplicate webhook delivery",
			mockResponseBody: []byte{},
		},
		{
			name:             "failure getting transaction",
			method:           http.MethodPost,
			status:           http.StatusUnauthorized,
			webhookPayload:   `{"data":{"id": "tx_1234_unauth", "amount": 2500}}`,
			logMessage:       "ERROR: problem getting transaction",
			mockResponseBody: []byte(`{"error":"Unauthorized"}`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			s.SweepThreshold = 1000
			// Set a mock response, if needed.
			if len(tc.mockResponseBody) > 0 {
				mockResponse(tc.status, map[string]string{"Content-Type": "application/json"}, tc.mockResponseBody)
			}
			// Use a faux logger so we can parse the content to find our debug logMessages to confirm our tests
			var fauxLog bytes.Buffer
			log.SetOutput(&fauxLog)
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.webhookPayload))
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(TxnHandler)
			handler.ServeHTTP(rr, req)
			if !strings.Contains(fauxLog.String(), tc.logMessage) {
				t.Errorf("'%v' failed.\nGot:\n%v\nExpected:\n%v", tc.name, fauxLog.String(), tc.logMessage)
			}
		})
	}
}
