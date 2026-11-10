#!/bin/sh

BACKEND_URL="https://captive-portal-ochre.vercel.app/api"
ROUTER_TOKEN="dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"
EXPIRY_FILE="/tmp/portal-expiry.txt"
LOCK_FILE="/var/lock/portal-sync.lock"

# single-instance lock
if [ -e "$LOCK_FILE" ]; then
    OLD_PID=$(cat "$LOCK_FILE" 2>/dev/null)
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
        exit 0
    fi
fi
echo $$ > "$LOCK_FILE"
trap 'rm -f "$LOCK_FILE"' EXIT INT TERM

now_epoch() { date +%s; }

while true; do
    RESPONSE=$(wget -q -T 15 -O - \
    --header "X-Router-Token: $ROUTER_TOKEN" \
    "$BACKEND_URL/check" 2>/dev/null)

    if [ -n "$RESPONSE" ]; then
        MACS_TO_ACK=""

        for MAC in $(echo "$RESPONSE" | tr -d ' \n' | grep -o '"mac":"[^"]*' | cut -d'"' -f4); do
            [ -z "$MAC" ] && continue

            DUR=$(echo "$RESPONSE" | tr -d ' \n' \
                  | grep -o "{\"mac\":\"$MAC\",\"duration\":\"[^\"]*" \
                  | grep -o '"duration":"[^"]*' | cut -d'"' -f4)
            [ -z "$DUR" ] && DUR=60

            if ndsctl auth "$MAC" >/dev/null 2>&1; then
                EXPIRY=$(( $(now_epoch) + DUR * 60 ))
                grep -v "^$MAC " "$EXPIRY_FILE" > "$EXPIRY_FILE.tmp" 2>/dev/null
                echo "$MAC $EXPIRY" >> "$EXPIRY_FILE.tmp"
                mv "$EXPIRY_FILE.tmp" "$EXPIRY_FILE"

                if [ -z "$MACS_TO_ACK" ]; then
                    MACS_TO_ACK="\"$MAC\""
                else
                    MACS_TO_ACK="$MACS_TO_ACK,\"$MAC\""
                fi
            fi
        done

        if [ -n "$MACS_TO_ACK" ]; then
            PAYLOAD="{\"macs\":[$MACS_TO_ACK]}"
            wget -q -T 15 -O /dev/null \
                --header "Content-Type: application/json" \
                --header "X-Router-Token: $ROUTER_TOKEN" \
                --post-data "$PAYLOAD" \
                "$BACKEND_URL/ack" 2>/dev/null
        fi
    fi

    # deauth expired clients
    if [ -f "$EXPIRY_FILE" ]; then
        NOW=$(now_epoch)
        while read -r MAC EXP; do
            [ -z "$MAC" ] && continue
            if [ "$NOW" -ge "$EXP" ]; then
                ndsctl deauth "$MAC" >/dev/null 2>&1
                grep -v "^$MAC " "$EXPIRY_FILE" > "$EXPIRY_FILE.tmp"
                mv "$EXPIRY_FILE.tmp" "$EXPIRY_FILE"
            fi
        done < "$EXPIRY_FILE"
    fi

    sleep 20
done