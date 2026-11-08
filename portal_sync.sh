#!/bin/sh
TOKEN="dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"
API_URL="https://captive-portal-ochre.vercel.app/"

RESPONSE=$(wget --no-check-certificate --header="X-Router-Token: $TOKEN" -qO- "$API_URL/check" 2>/dev/null)

if [ -z "$RESPONSE" ] || [ "$RESPONSE" = '{"unlock":[]}' ] || [ "$RESPONSE" = '{"unlock":null}' ]; then 
    exit 0 
fi

ACK_MACS=""
INDEX=0

while true; do
    MAC=$(echo "$RESPONSE" | jsonfilter -e "@.unlock[$INDEX].mac")
    DURATION=$(echo "$RESPONSE" | jsonfilter -e "@.unlock[$INDEX].duration")

    if [ -z "$MAC" ]; then
        break
    fi

    if [ -z "$DURATION" ]; then 
        DURATION=60
    fi
    
    SLEEP_SECONDS=$((DURATION * 60))

    iptables -C captive_filter -m mac --mac-source "$MAC" -j ACCEPT 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "[+] Whitelisting Subscriber MAC: $MAC for $DURATION minutes"
        iptables -I captive_filter 1 -m mac --mac-source "$MAC" -j ACCEPT
        (sleep $SLEEP_SECONDS && iptables -D captive_filter -m mac --mac-source "$MAC" -j ACCEPT) &
    fi

    if [ -z "$ACK_MACS" ]; then 
        ACK_MACS="\"$MAC\""
    else 
        ACK_MACS="$ACK_MACS,\"$MAC\""
    fi
    
    INDEX=$((INDEX + 1))
done

if [ ! -z "$ACK_MACS" ]; then
    wget --no-check-certificate \
         --header="X-Router-Token: $TOKEN" \
         --header="Content-Type: application/json" \
         --post-data="{\"macs\":[$ACK_MACS]}" \
         -qO- "$API_URL/ack" >/dev/null 2>&1
fi