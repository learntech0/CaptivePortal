#!/bin/sh
TOKEN="dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"
API_URL="https://captive-portal-ochre.vercel.app/api/check"

RESPONSE=$(wget --no-check-certificate --header="X-Router-Token: $TOKEN" -qO- "$API_URL" 2>/dev/null)

# Ensure response is not empty
if [ -z "$RESPONSE" ]; then
    exit 0
fi

# Loop through the authorized addresses (JSON objects)
# Using jsonfilter to extract each object's MAC and duration
echo "$RESPONSE" | jsonfilter -e '@.unlock[*]' | while read -r DEVICE; do
    MAC=$(echo "$DEVICE" | jsonfilter -e '@.mac')
    DURATION=$(echo "$DEVICE" | jsonfilter -e '@.duration')
    
    # Default to 60 minutes if duration is missing (safety)
    if [ -z "$DURATION" ]; then DURATION=60; fi
    
    # Convert minutes to seconds for sleep
    SLEEP_SECONDS=$((DURATION * 60))

    iptables -C forwarding_rule -m mac --mac-source "$MAC" -j ACCEPT 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "[+] Whitelisting Subscriber MAC: $MAC for $DURATION minutes"
        iptables -I forwarding_rule 1 -m mac --mac-source "$MAC" -j ACCEPT

        # Schedule a background self-destruct with the CORRECT duration
        (sleep $SLEEP_SECONDS && iptables -D forwarding_rule -m mac --mac-source "$MAC" -j ACCEPT) &
    fi
done