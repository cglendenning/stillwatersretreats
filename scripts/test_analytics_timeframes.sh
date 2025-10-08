#!/bin/bash

# Enhanced test script to generate realistic analytics data across multiple time periods
# Usage: ./test_analytics_timeframes.sh [--size small|med|large]

echo "🎯 Generating test analytics data across multiple timeframes..."
echo ""

BASE_URL="http://localhost:8080"

# Parse size parameter
SIZE="small"
if [ "$1" = "--size" ]; then
    SIZE="${2:-small}"
fi

# Set scale multiplier based on size
case $SIZE in
    small)
        SCALE=1
        ;;
    med)
        SCALE=10
        ;;
    large)
        SCALE=100
        ;;
    *)
        echo "❌ Invalid size. Use: small, med, or large"
        exit 1
        ;;
esac

echo "📏 Size: $SIZE (${SCALE}x scale)"
echo ""

# Function to pick a weighted random route
# Routes are weighted by realistic traffic patterns
pick_route() {
    local rand=$((RANDOM % 263))  # Total weight = 263
    
    # Weighted selection based on cumulative weights
    if [ $rand -lt 100 ]; then echo "/"; return; fi                      # 100/263 = 38%
    rand=$((rand - 100))
    if [ $rand -lt 35 ]; then echo "/bearviewcove"; return; fi           # 35/263 = 13%
    rand=$((rand - 35))
    if [ $rand -lt 30 ]; then echo "/victoriapines"; return; fi          # 30/263 = 11%
    rand=$((rand - 30))
    if [ $rand -lt 18 ]; then echo "/cabins"; return; fi                 # 18/263 = 7%
    rand=$((rand - 18))
    if [ $rand -lt 15 ]; then echo "/pricing"; return; fi                # 15/263 = 6%
    rand=$((rand - 15))
    if [ $rand -lt 12 ]; then echo "/calendar"; return; fi               # 12/263 = 5%
    rand=$((rand - 12))
    if [ $rand -lt 10 ]; then echo "/activities"; return; fi             # 10/263 = 4%
    rand=$((rand - 10))
    if [ $rand -lt 8 ]; then echo "/howitworks"; return; fi              # 8/263 = 3%
    rand=$((rand - 8))
    if [ $rand -lt 7 ]; then echo "/contact"; return; fi                 # 7/263 = 3%
    rand=$((rand - 7))
    if [ $rand -lt 6 ]; then echo "/packages"; return; fi                # 6/263 = 2%
    rand=$((rand - 6))
    if [ $rand -lt 5 ]; then echo "/blog"; return; fi                    # 5/263 = 2%
    rand=$((rand - 5))
    if [ $rand -lt 4 ]; then echo "/coffee"; return; fi                  # 4/263 = 2%
    rand=$((rand - 4))
    if [ $rand -lt 3 ]; then echo "/crossfit"; return; fi                # 3/263 = 1%
    rand=$((rand - 3))
    if [ $rand -lt 3 ]; then echo "/coaching"; return; fi                # 3/263 = 1%
    rand=$((rand - 3))
    if [ $rand -lt 2 ]; then echo "/coachcraig"; return; fi              # 2/263 = 1%
    rand=$((rand - 2))
    if [ $rand -lt 2 ]; then echo "/coaching-detail"; return; fi         # 2/263 = 1%
    rand=$((rand - 2))
    if [ $rand -lt 2 ]; then echo "/review"; return; fi                  # 2/263 = 1%
    rand=$((rand - 2))
    echo "/success"                                                       # 1/263 = 0.4%
}

# Function to track an event with a specific timestamp
track_event_at_time() {
    local event_type=$1
    local page=$2
    local cabin=$3
    local amount=$4
    local days_ago=$5
    local hours_ago=$6
    
    # Calculate timestamp
    local timestamp=$(date -v-${days_ago}d -v-${hours_ago}H -u +"%Y-%m-%dT%H:%M:%S-07:00")
    
    # Generate random user agent to simulate different visitors
    # Scale unique visitors with SCALE to maintain realistic conversion rates
    local visitor_pool=$((100 * SCALE))
    local browser_num=$((RANDOM % visitor_pool + 1))
    local user_agent="Mozilla/5.0 (TestVisitor${browser_num})"
    
    # Create event with timestamp
    local event_json=$(cat <<EOF
{
  "timestamp": "${timestamp}",
  "event_type": "${event_type}",
  "page": "${page}",
  "cabin": "${cabin}",
  "amount": "${amount}",
  "user_agent": "${user_agent}"
}
EOF
)
    
    # Append to assets/data/analytics.json using jq
    if [ -f ../assets/data/analytics.json ]; then
        # Use jq to properly append to the array
        echo "$event_json" | jq -s '.[0]' | jq --slurpfile existing ../assets/data/analytics.json '$existing[0] + [.]' > temp.json && mv temp.json ../assets/data/analytics.json
    else
        # Create new file with single event
        echo "[$event_json]" | jq '.' > ../assets/data/analytics.json
    fi
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "❌ This script requires 'jq' to be installed."
    echo "Install it with: brew install jq"
    exit 1
fi

# Backup existing analytics
if [ -f ../assets/data/analytics.json ]; then
    cp ../assets/data/analytics.json ../assets/data/analytics.backup.json
    echo "📦 Backed up existing analytics to assets/data/analytics.backup.json"
fi

# Initialize empty analytics file
echo "[]" > ../assets/data/analytics.json

echo "📅 Generating data for LAST 45 DAYS..."
echo ""

# Calculate event counts based on scale
PV_OLD=$((10 * SCALE))           # 10/100/1000 page views 40-45 days ago
PV_30=$((20 * SCALE))            # 20/200/2000 page views 25-30 days ago
PV_14=$((30 * SCALE))            # 30/300/3000 page views 8-14 days ago
PV_7=$((50 * SCALE))             # 50/500/5000 page views 2-7 days ago
PV_TODAY=$((80 * SCALE))         # 80/800/8000 page views last 24h

CS_OLD=$((1 * SCALE))            # Cabin selections
CS_30=$((10 * SCALE))
CS_14=$((15 * SCALE))
CS_7=$((25 * SCALE))
CS_TODAY=$((40 * SCALE))

BA_30=$((5 * SCALE))             # Booking attempts
BA_14=$((8 * SCALE))
BA_7=$((12 * SCALE))
BA_TODAY=$((20 * SCALE))

CO_30=$((0 * SCALE))             # Checkouts (reduced for realistic ~2-4% conversion rate)
CO_14=$((1 * SCALE))
CO_7=$((1 * SCALE))
CO_TODAY=$((2 * SCALE))

# === 40-45 DAYS AGO ===
echo "⏰ Creating events from 40-45 days ago..."
for ((i=1; i<=PV_OLD; i++)); do
    route=$(pick_route)
    track_event_at_time "page_view" "$route" "" "" "$((RANDOM % 6 + 40))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CS_OLD; i++)); do
    track_event_at_time "cabin_selection" "/" "Bear View Cove" "" "43" "$(($RANDOM % 24))"
done
track_event_at_time "checkout_session" "/" "Bear View Cove" "\$899" "43" "14"

# === 25-30 DAYS AGO ===
echo "⏰ Creating events from 25-30 days ago..."
for ((i=1; i<=PV_30; i++)); do
    route=$(pick_route)
    track_event_at_time "page_view" "$route" "" "" "$((RANDOM % 6 + 25))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CS_30; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    track_event_at_time "cabin_selection" "/" "$cabin" "" "$((RANDOM % 6 + 25))" "$(($RANDOM % 24))"
done
for ((i=1; i<=BA_30; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "booking_attempt" "/review" "$cabin" "$amount" "$((RANDOM % 6 + 25))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CO_30; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "checkout_session" "/review" "$cabin" "$amount" "$((RANDOM % 6 + 25))" "$(($RANDOM % 24))"
done

# === 8-14 DAYS AGO ===
echo "⏰ Creating events from 8-14 days ago..."
for ((i=1; i<=PV_14; i++)); do
    route=$(pick_route)
    track_event_at_time "page_view" "$route" "" "" "$((RANDOM % 7 + 8))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CS_14; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    track_event_at_time "cabin_selection" "/" "$cabin" "" "$((RANDOM % 7 + 8))" "$(($RANDOM % 24))"
done
for ((i=1; i<=BA_14; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "booking_attempt" "/review" "$cabin" "$amount" "$((RANDOM % 7 + 8))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CO_14; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "checkout_session" "/review" "$cabin" "$amount" "$((RANDOM % 7 + 8))" "$(($RANDOM % 24))"
done
track_event_at_time "contact_form" "/contact" "" "" "11" "16"

# === 2-7 DAYS AGO ===
echo "⏰ Creating events from 2-7 days ago..."
for ((i=1; i<=PV_7; i++)); do
    route=$(pick_route)
    track_event_at_time "page_view" "$route" "" "" "$((RANDOM % 6 + 2))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CS_7; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    track_event_at_time "cabin_selection" "/" "$cabin" "" "$((RANDOM % 6 + 2))" "$(($RANDOM % 24))"
done
for ((i=1; i<=BA_7; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "booking_attempt" "/review" "$cabin" "$amount" "$((RANDOM % 6 + 2))" "$(($RANDOM % 24))"
done
for ((i=1; i<=CO_7; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "checkout_session" "/review" "$cabin" "$amount" "$((RANDOM % 6 + 2))" "$(($RANDOM % 24))"
done
for ((i=1; i<=$((2 * SCALE)); i++)); do
    track_event_at_time "contact_form" "/contact" "" "" "$((RANDOM % 6 + 2))" "$(($RANDOM % 24))"
done

# === TODAY (Last 24 hours) ===
echo "⏰ Creating events from last 24 hours (TODAY)..."
for ((i=1; i<=PV_TODAY; i++)); do
    route=$(pick_route)
    track_event_at_time "page_view" "$route" "" "" "0" "$(($RANDOM % 24))"
done
for ((i=1; i<=CS_TODAY; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    track_event_at_time "cabin_selection" "/" "$cabin" "" "0" "$(($RANDOM % 24))"
done
for ((i=1; i<=BA_TODAY; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "booking_attempt" "/review" "$cabin" "$amount" "0" "$(($RANDOM % 24))"
done
for ((i=1; i<=CO_TODAY; i++)); do
    cabin=$( [ $((RANDOM % 2)) -eq 0 ] && echo "Bear View Cove" || echo "Victoria Pines" )
    amount=$( [ "$cabin" = "Bear View Cove" ] && echo "\$899" || echo "\$950" )
    track_event_at_time "checkout_session" "/review" "$cabin" "$amount" "0" "$(($RANDOM % 24))"
done
for ((i=1; i<=$((3 * SCALE)); i++)); do
    track_event_at_time "contact_form" "/contact" "" "" "0" "$(($RANDOM % 24))"
done

echo ""
echo "✨ Test data generation complete!"
echo ""
echo "📊 Summary by event type:"
cat ../assets/data/analytics.json | jq '. | group_by(.event_type) | map({event: .[0].event_type, count: length}) | .[]'
echo ""
echo "📊 Top 10 routes by traffic:"
cat ../assets/data/analytics.json | jq -r '.[] | select(.event_type == "page_view") | .page' | sort | uniq -c | sort -rn | head -10
echo ""
echo "🎯 Total events: $(cat ../assets/data/analytics.json | jq '. | length')"
echo ""
echo "📏 Scale used: $SIZE (${SCALE}x multiplier)"
echo ""
echo "🔍 Now test your dashboard time filters:"
echo "   1. Open: ${BASE_URL}/dashboard"
echo "   2. Click '24 Hours' to see today's traffic"
echo "   3. Click '7 Days' to see weekly trends"
echo "   4. Click '30 Days' to see monthly patterns"
echo "   5. Click 'All Time' to see everything"
echo ""
echo "📈 Watch routes, charts, and funnel change between time periods!"
echo ""
echo "💾 To restore original data: mv ../assets/data/analytics.backup.json ../assets/data/analytics.json"
