package backend

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func getEnvOrFail(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env var: " + key)
	}
	return v
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid payload"})
		return
	}

	// server-side validation
	if !isValidPhone(req.Phone) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid phone number"})
		return
	}
	if !isValidMAC(req.Mac) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid MAC address"})
		return
	}
	if !isValidRouterIP(req.RouterIp) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid router IP"})
		return
	}

	// 60s duplicate-payment guard
	InitRedis()
	dedupKey := fmt.Sprintf("register:%s:%v:%d", req.Mac, req.Amount, time.Now().Unix()/60)
	wasSet, _ := rdb.SetNX(ctx, dedupKey, "1", 60*time.Second).Result()
	if !wasSet {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "A payment request is already in progress. Please wait."})
		return
	}

	var durationMinutes int
	switch req.Amount {
	case 10:
		durationMinutes = 60
	case 20:
		durationMinutes = 180
	case 50:
		durationMinutes = 1440
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported subscription package tier selected."})
		return
	}

	// fail loud if env missing
	consumerKey    := getEnvOrFail("MPESA_CONSUMER_KEY")
	consumerSecret := getEnvOrFail("MPESA_CONSUMER_SECRET")
	authURL        := getEnvOrFail("MPESA_AUTH_URL")
	stkPushURL     := getEnvOrFail("MPESA_STK_PUSH_URL")
	callbackURL    := getEnvOrFail("MPESA_CALLBACK_URL")
	shortcode      := getEnvOrFail("MPESA_BUSINESS_TILL_NUMBER")
	passkey        := getEnvOrFail("MPESA_PASSKEY")

	// Request budget: 18s total, well under Vercel limits
	ctxReq, cancel := context.WithTimeout(r.Context(), 18*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 8 * time.Second}

	// --- 1. Access token ---
	authCredentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
	tokenReq, _ := http.NewRequestWithContext(ctxReq, "GET", authURL, nil)
	tokenReq.Header.Set("Authorization", "Basic "+authCredentials)

	tokenRes, err := client.Do(tokenReq)
	if err != nil {
		logf("register: token request failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to connect to authentication gateway"})
		return
	}
	defer tokenRes.Body.Close()

	var tokenData TokenResponse
	if err := json.NewDecoder(tokenRes.Body).Decode(&tokenData); err != nil || tokenData.AccessToken == "" {
		logf("register: token decode failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication structure parse failure"})
		return
	}

	// --- 2. STK push ---
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(shortcode + passkey + timestamp))

	formattedPhone := req.Phone
	if strings.HasPrefix(formattedPhone, "0") {
		formattedPhone = "254" + formattedPhone[1:]
	}

	// Short opaque ref: MAC with colons stripped + duration.
	// e.g. "AABBCCDDEEFF|1440" — 15 chars, safe for Safaricom's truncation.
	accountRef := fmt.Sprintf("%s|%d", strings.ReplaceAll(req.Mac, ":", ""), durationMinutes)

	stkPayload := map[string]interface{}{
		"BusinessShortCode": shortcode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerBuyGoodsOnline",
		"Amount":            req.Amount,
		"PartyA":            formattedPhone,
		"PartyB":            shortcode,
		"PhoneNumber":       formattedPhone,
		"CallBackURL":       callbackURL,
		"AccountReference":  accountRef,
		"TransactionDesc":   "WiFi Premium High Speed Access Token",
	}

	jsonBytes, _ := json.Marshal(stkPayload)
	stkReq, _ := http.NewRequestWithContext(ctxReq, "POST", stkPushURL, bytes.NewBuffer(jsonBytes))
	stkReq.Header.Set("Content-Type", "application/json")
	stkReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	stkRes, err := client.Do(stkReq)
	if err != nil {
		logf("register: stk push failed mac=%s amount=%v: %v", req.Mac, req.Amount, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to invoke target transaction execution"})
		return
	}
	defer stkRes.Body.Close()

	logf("register: stk pushed mac=%s amount=%v duration=%d", req.Mac, req.Amount, durationMinutes)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "STK push triggered successfully!"})
}