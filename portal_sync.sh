#!/bin/sh
TOKEN="dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"
API_URL="https://vercel.app"

# 1. Fetch pending devices
RESPONSE=$(wget --no-check-certificate --header="X-Router-Token: $TOKEN" -qO- "$API_URL/check" 2>/dev/null)

# Exit if response is empty or the queue is empty
if [ -z "$RESPONSE" ] || [ "$RESPONSE" = '{"unlock":[]}' ] || [ "$RESPONSE" = '{"unlock":null}' ]; then 
    exit 0 
fi

ACK_MACS=""
INDEX=0

# 2. Loop through the array using jsonfilter by index
while true; do
    MAC=$(echo "$RESPONSE" | jsonfilter -e "@.unlock[$INDEX].mac")
    DURATION=$(echo "$RESPONSE" | jsonfilter -e "@.unlock[$INDEX].duration")

    # Break out of loop when no more elements are found
    if [ -z "$MAC" ]; then
        break
    fi

    if [ -z "$DURATION" ]; then 
        DURATION=60
    fi
    
    SLEEP_SECONDS=$((DURATION * 60))

    # FIXED: Check if rule already exists inside captive_filter instead of forwarding_rule
    iptables -C captive_filter -m mac --mac-source "$MAC" -j ACCEPT 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "[+] Whitelisting Subscriber MAC: $MAC for $DURATION minutes"
        # FIXED: Insert rule at position 1 of captive_filter so it evaluates before the global DROP
        iptables -I captive_filter 1 -m mac --mac-source "$MAC" -j ACCEPT

        # FIXED: Schedule async expiration to delete from captive_filter when time runs out
        (sleep $SLEEP_SECONDS && iptables -D captive_filter -m mac --mac-source "$MAC" -j ACCEPT) &
    fi

    # Build the ACK string (e.g. "AA:BB:CC,DD:EE:FF")
    if [ -z "$ACK_MACS" ]; then 
        ACK_MACS="\"$MAC\""
    else 
        ACK_MACS="$ACK_MACS,\"$MAC\""
    fi
    
    INDEX=$((INDEX + 1))
done

# 3. Explicitly ACK these items back to Vercel to wipe them from Redis safely
if [ ! -z "$ACK_MACS" ]; then
    wget --no-check-certificate \
         --header="X-Router-Token: $TOKEN" \
         --header="Content-Type: application/json" \
         --post-data="{\"macs\":[$ACK_MACS]}" \
         -qO- "$API_URL/ack" >/dev/null 2>&1
fi
