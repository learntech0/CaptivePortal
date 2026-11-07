package backend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type PayRequest struct {
	Phone    string  `json:"phone"`
	Amount   float64 `json:"amount"`
	Mac      string  `json:"mac"`
	RouterIp string  `json:"router_ip"`
}

// Safaricom OAuth response struct
type TokenResponse struct {
	AccessToken string `json:"access_token"`
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

	// Determine time packages
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

	// Fetch M-PESA variables safely from Vercel environment
	consumerKey := os.Getenv("MPESA_CONSUMER_KEY")
	consumerSecret := os.Getenv("MPESA_CONSUMER_SECRET")
	authURL := os.Getenv("MPESA_AUTH_URL")
	stkPushURL := os.Getenv("MPESA_STK_PUSH_URL")

	// Fallbacks matching your development credentials if local environment is missing variables
	if consumerKey == "" {
		consumerKey = "o6yKcsA2KxV54AjDTa1VgMd4ZcqwxfCvAzrbRCnkqOtZDOdt"
		consumerSecret = "3nrbJhDxP9Gvw0BXvteqHJBYBBVfryCrJXN3oHljGwgXF3pOmAAJKaVf1NEwC8St"
		authURL = "https://safaricom.co.ke"
		stkPushURL = "https://safaricom.co.ke"
	}

	// 1. Generate M-Pesa Access Token
	authCredentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
	tokenReq, _ := http.NewRequest("GET", authURL, nil)
	tokenReq.Header.Set("Authorization", "Basic "+authCredentials)

	client := &http.Client{Timeout: 10 * time.Second}
	tokenRes, err := client.Do(tokenReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to connect to authentication gateway"})
		return
	}
	defer tokenRes.Body.Close()

	var tokenData TokenResponse
	if err := json.NewDecoder(tokenRes.Body).Decode(&tokenData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication structure parse failure"})
		return
	}

	// 2. Build M-Pesa STK Push Parameters
	shortcode := "174379" // Default Lipa Na M-Pesa sandbox shortcode
	passkey := "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(shortcode + passkey + timestamp))

	// Clean up phone number serialization format (07... to 2547...)
	formattedPhone := req.Phone
	if strings.HasPrefix(formattedPhone, "0") {
		formattedPhone = "254" + formattedPhone[1:]
	}

	// Pack tracking reference tags inside metadata
	accountRef := fmt.Sprintf("%s|%s|%d", req.Mac, req.RouterIp, durationMinutes)

	stkPayload := map[string]interface{}{
		"BusinessShortCode": shortcode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerBuyGoodsOnline", // Custom configuration preferred by you
		"Amount":            req.Amount,
		"PartyA":            formattedPhone,
		"PartyB":            shortcode,
		"PhoneNumber":       formattedPhone,
		"CallBackURL":       "https://vercel.app", // Target dynamic hook gateway
		"AccountReference":  accountRef,
		"TransactionDesc":   "WiFi Premium High Speed Access Token",
	}

	// 3. Dispatch payload request to Safaricom Daraja API Engine
	jsonBytes, _ := json.Marshal(stkPayload)
	stkReq, _ := http.NewRequest("POST", stkPushURL, bytes.NewBuffer(jsonBytes))
	stkReq.Header.Set("Content-Type", "application/json")
	stkReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	stkRes, err := client.Do(stkReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to invoke target transaction execution"})
		return
	}
	defer stkRes.Body.Close()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "STK push triggered successfully!"})
}
