#!/bin/sh
TOKEN="dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"
API_URL="https://captive-portal-ochre.vercel.app/api/check"

# Fetch paid MAC addresses securely from the cloud backend
RESPONSE=$(wget --header="X-Router-Token: $TOKEN" -qO- "$API_URL" 2>/dev/null)

# Loop through the authorized addresses returned by Vercel
for MAC in $(echo "$RESPONSE" | jsonfilter -e '@.unlock[*]'); do
    # Check if the firewall rule already exists for this user to avoid duplication
    iptables -C forwarding_rule -m mac --mac-source "$MAC" -j ACCEPT 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "[+] Whitelisting Subscriber MAC: $MAC"
        iptables -I forwarding_rule 1 -m mac --mac-source "$MAC" -j ACCEPT

        # Schedule a background self-destruct to remove access after 1 hour
        (sleep 3600 && iptables -D forwarding_rule -m mac --mac-source "$MAC" -j ACCEPT) &
    fi
done
