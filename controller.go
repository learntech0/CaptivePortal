// Run locally: go run controller.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

const MAX_SUBSCRIBERS = 15
const SHARED_SECRET = "dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"

type ControlPayload struct {
	MacAddress      string `json:"macAddress"`
	RouterIp        string `json:"routerIp"`
	DurationMinutes string `json:"durationMinutes"`
	Secret          string `json:"secret"`
}

// Thread-safe map and tracking systems for our test cluster limits
var activeSubscribers = make(map[string]bool)
var mu sync.Mutex

func processAccessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ControlPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Security handshake guard
	if payload.Secret != SHARED_SECRET {
		http.Error(w, "Unauthorized payload transaction", http.StatusForbidden)
		return
	}

	minutes, err := strconv.Atoi(payload.DurationMinutes)
	if err != nil {
		http.Error(w, "Invalid duration formatting", http.StatusBadRequest)
		return
	}

	mu.Lock()
	// Guard against exceeding backhaul speed allowance limits
	if len(activeSubscribers) >= MAX_SUBSCRIBERS && !activeSubscribers[payload.MacAddress] {
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Capacity limit hit on local node backhaul."})
		return
	}

	// Register or renew user capacity footprint 
	activeSubscribers[payload.MacAddress] = true
	currentTotalCount := len(activeSubscribers)
	mu.Unlock()

	fmt.Printf("[+] Authorizing MAC: %s on Router %s for %d Mins. Total active: %d/%d\n", 
		payload.MacAddress, payload.RouterIp, minutes, currentTotalCount, MAX_SUBSCRIBERS)

	// Inject the dynamic iptables whitelist rule
	injectFirewallRule(payload.RouterIp, payload.MacAddress, "-I")

	// EXPRIATION AUTOMATION TIMER: Spin up a thread-safe background countdown hook
	time.AfterFunc(time.Duration(minutes)*time.Minute, func() {
		mu.Lock()
		delete(activeSubscribers, payload.MacAddress)
		remainingCount := len(activeSubscribers)
		mu.Unlock()

		fmt.Printf("[-] Session Expired for MAC: %s on Router %s. Total remaining: %d\n", 
			payload.MacAddress, payload.RouterIp, remainingCount)

		// Delete the client rule from the device kernel memory
		injectFirewallRule(payload.RouterIp, payload.MacAddress, "-D")
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Access granted successfully"})
}

func injectFirewallRule(routerIp, macAddress, action string) {
	// 1. Target the correct OpenWrt system user
	username := "root" 
	password := "tp_link" 

	// 2. Build the precise command text to inject directly into the router's kernel memory
	// This command inserts (-I) or deletes (-D) the MAC whitelist rule inside the OpenWrt forwarding chain
	cmdString := fmt.Sprintf("iptables %s forwarding_rule 1 -m mac --mac-source %s -j ACCEPT", action, macAddress)
	
	// 3. Use sshpass to log in automatically without waiting for a manual password prompt
	cmd := exec.Command("sshpass", "-p", password, "ssh", "-o", "StrictHostKeyChecking=no", username+"@"+routerIp, cmdString)
	
	// 4. Fire the command
	if err := cmd.Run(); err != nil {
		log.Printf("[!] Local Controller Firewall Error (%s action on MAC %s): %v\n", action, macAddress, err)
	} else {
		log.Printf("[*] Successfully sent %s rule for MAC %s down to Router %s\n", action, macAddress, routerIp)
	}
}


func main() {
	http.HandleFunc("/process-access", processAccessHandler)
	fmt.Println("[*] Local Network Controller online on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
