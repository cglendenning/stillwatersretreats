#!/bin/bash

# Add tracking to all template pages that don't have it yet

cd /Users/craig/stillwaters/templates

TRACKING='<script>\
function trackEvent(eventType, page, cabin, amount) {\
    fetch('\''/api/track'\'', {\
        method: '\''POST'\'',\
        headers: { '\''Content-Type'\'': '\''application/json'\'' },\
        body: JSON.stringify({ event_type: eventType, page: page || window.location.pathname, cabin: cabin || '\'\'\'', amount: amount || '\'''\'' })\
    }).catch(err => console.log('\''Tracking error:'\'', err));\
}\
window.addEventListener('\''load'\'', function() { trackEvent('\''page_view'\'', window.location.pathname); });\
</script>'

# Pages to process (excluding nav, footer, dashboard, etc.)
PAGES=(
    "blog_list.html"
    "blog_template.html"
    "book-bearview.html"
    "booking.html"
    "calendar.html"
    "cancel.html"
    "coachcraig.html"
    "coaching-detail.html"
    "coaching.html"
    "coffee.html"
    "crossfit.html"
    "email-sent.html"
    "howitworks.html"
    "packages.html"
    "review.html"
    "success.html"
)

for page in "${PAGES[@]}"; do
    if [ -f "$page" ]; then
        if grep -q "trackEvent" "$page"; then
            echo "✓ $page already has tracking"
        else
            # Find </head> and insert before it using perl (more reliable than sed for multiline)
            perl -i -pe "s{</head>}{$TRACKING\n</head>}" "$page"
            echo "✓ Added tracking to $page"
        fi
    else
        echo "✗ $page not found"
    fi
done

echo ""
echo "✅ Done! All pages now have tracking."

