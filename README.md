# Still Waters Retreats

Deep thinking retreats at Big Bear Lake, CA for creatives, builders, and deep thinkers who need uninterrupted time to focus, create, and recharge.

## Quick Start

```bash
# Run the server
go run main.go

# Or build and run
go build -o stillwaters main.go
./stillwaters
```

Site runs at `http://localhost:8080`

## Tech Stack

- **Backend**: Go (Echo framework)
- **Frontend**: Static HTML/CSS with Vue.js components
- **Payments**: Stripe
- **Calendar**: Hospitable API
- **Email**: Amazon SES
- **Analytics**: Firebase + Custom Dashboard

## Key Integrations

- **Stripe**: Payment processing
- **Hospitable**: Property/calendar management
- **Google reCAPTCHA**: Form protection
- **Amazon SES**: Transactional emails

## Project Structure

```
├── main.go              # Go server & API
├── templates/           # HTML templates
├── assets/
│   ├── data/            # Analytics & data files
│   ├── photo/           # Images
│   └── video/           # Videos
├── blog/                # Blog posts (JSON)
├── static/              # CSS, JS
└── scripts/             # Test & utility scripts
```

## Analytics Dashboard

**Access**: `http://localhost:8080/dashboard`  
**Password**: Set `DASHBOARD_PASSWORD` in `.env` (default: `stillwaters2024`)

### What It Tracks

- Page views by route
- Unique visitors
- Cabin selections
- Booking attempts
- Checkout sessions
- Conversion funnel
- Time-based trends

### Features

- **Route Tracking**: See which pages get the most traffic
- **Conversion Funnel**: Track user journey from visit to checkout
- **Time Filtering**: View 24h, 7d, 30d, or all-time metrics
- **Charts**: Traffic trends, route distribution, cabin interest
- **Auto-refresh**: Updates every 60 seconds

### Data Storage

All analytics stored in `analytics.json` (last 1,000 events). Add to `.gitignore`:

```
analytics.json
```

## Environment Variables

Create a `.env` file:

```bash
# Dashboard
DASHBOARD_PASSWORD=your_password

# AWS/SES
AWS_REGION=us-east-1
SES_FROM_EMAIL=your_email@domain.com
SES_TO_EMAIL=your_email@domain.com

# Stripe
STRIPE_SECRET_KEY=your_stripe_key

# Hospitable
HOSPITABLE_API_KEY=your_hospitable_key

# reCAPTCHA
RECAPTCHA_SITE_KEY=your_site_key
RECAPTCHA_API_KEY=your_api_key
RECAPTCHA_PROJECT_ID=your_project_id

# Firebase
FIREBASE_API_KEY=your_firebase_key

# Environment
ENVIRONMENT=local  # or leave blank for production
```

## AWS SES Email Setup

### Quick Setup

1. **Verify your domain in SES** (us-east-1)
2. **Request production access** (exit sandbox mode)
3. **Create IAM role** for EC2 with `AmazonSESFullAccess` policy
4. **Set environment variables:**

```bash
AWS_REGION=us-east-1
SES_FROM_EMAIL=noreply@yourdomain.com
SES_TO_EMAIL=your_email@yourdomain.com
```

### Key Points

- **Don't send FROM third-party emails** (yahoo.com, gmail.com) - DMARC policies will block
- **Send FROM your domain** - Use Reply-To for easy responses
- **On EC2**: IAM role provides credentials automatically (no keys needed)
- **Local dev**: Set `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in `.env`

### Email Flow

```
Contact form → Go backend → SES → Your email
Reply → Customer email (via Reply-To header)
```

## Deployment

Automated via GitHub Actions to Ubuntu server with Nginx.

## Philosophy

Environment is everything for deep work. These cabins and this platform help you reclaim focus, find clarity, and do your best work.

---

Visit [stillwatersretreats.com](https://stillwatersretreats.com)
