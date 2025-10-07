# Amazon SES Setup Guide with IAM Role

## Quick Summary of the Issue

**Problem:** You were trying to send email FROM `c_glendenning@yahoo.com` through Amazon SES. Yahoo has strict DMARC policies that block this to prevent email spoofing.

**Solution:** Send FROM a domain you own and control (e.g., `noreply@stillwatersretreats.com`), and use Reply-To to make it easy to respond to customers.

---

## Step 1: Install AWS SDK Dependencies on Your Server

SSH into your EC2 instance and run:

```bash
cd ~/stillwaters-website
go get github.com/aws/aws-sdk-go-v2/aws
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/aws/aws-sdk-go-v2/service/sesv2
go mod tidy
```

---

## Step 2: Create IAM Role with SES Permissions

### 2.1 Create the Policy

1. Go to **AWS Console** → **IAM** → **Policies**
2. Click **Create policy** → **JSON** tab
3. Paste this policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ses:SendEmail",
        "ses:SendRawEmail"
      ],
      "Resource": "*"
    }
  ]
}
```

4. Click **Next**
5. Name: `SESEmailSendingPolicy`
6. Click **Create policy**

### 2.2 Create the IAM Role

1. Go to **IAM** → **Roles** → **Create role**
2. **Trusted entity type**: AWS service
3. **Use case**: Select **EC2** (or your compute service)
4. Click **Next**
5. Search for and select `SESEmailSendingPolicy`
6. Click **Next**
7. Role name: `stillwaters-ses-role`
8. Click **Create role**

### 2.3 Attach Role to Your EC2 Instance

1. Go to **EC2** → **Instances**
2. Select your instance
3. **Actions** → **Security** → **Modify IAM role**
4. Select `stillwaters-ses-role`
5. Click **Update IAM role**

---

## Step 3: Verify Your Domain in SES

### Option A: If You Have a Domain (Recommended)

1. Go to **Amazon SES** → **Verified identities**
2. Click **Create identity**
3. Select **Domain**
4. Enter your domain: `stillwatersretreats.com` (without www)
5. Check **Use a default MAIL FROM domain** (optional but recommended)
6. Click **Create identity**
7. You'll see DNS records to add. Add these to your domain's DNS:
   - Usually 3 CNAME records for DKIM
   - 1 TXT record for domain verification
   - (Optional) MX record for MAIL FROM
8. Wait 10-30 minutes for verification

### Option B: If You Don't Have a Domain Yet

1. For now, verify just an email address:
   - Go to **Amazon SES** → **Verified identities**
   - Click **Create identity** → Select **Email address**
   - Enter your email (must be one you control)
   - Check your email and click the verification link

**Important:** With just a verified email, you can only send FROM that exact email address. To send from any address on your domain (like `noreply@yourdomain.com`), you must verify the entire domain.

---

## Step 4: Request Production Access (Important!)

By default, SES is in "Sandbox" mode - you can only send TO verified email addresses.

1. Go to **Amazon SES** → **Account dashboard**
2. Look for the banner about "Sandbox" mode
3. Click **Request production access**
4. Fill out the form:
   - **Use case**: Transactional emails for a retreat booking website
   - **Compliance**: Explain you're sending booking confirmations and contact form notifications
   - **Rate**: Start with something reasonable like 50 emails/day
5. Submit - usually approved within 24 hours

---

## Step 5: Update Environment Variables

Update your environment variables (in your `.env` file or EC2 environment):

```bash
# AWS Region (where your SES is set up)
AWS_REGION=us-east-1

# Email Configuration
# FROM: Must be verified in SES (use your domain!)
SES_FROM_EMAIL=noreply@stillwatersretreats.com
# TO: Where you want to receive contact form emails
SES_TO_EMAIL=c_glendenning@yahoo.com

# DO NOT SET THESE IN PRODUCTION (IAM role handles auth automatically):
# AWS_ACCESS_KEY_ID=
# AWS_SECRET_ACCESS_KEY=
```

**For Local Development Only:**
If you want to test locally, create an IAM user with access keys and add to your local `.env`:
```bash
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
```

---

## Step 6: Remove Old SMTP Environment Variables

You can remove these from your environment (they're no longer used):
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM_EMAIL`
- `SMTP_TO_EMAIL`

---

## Step 7: Deploy and Test

1. Commit and push your changes:
```bash
git add .
git commit -m "Switch from SMTP to SES v2 API with IAM role"
git push
```

2. On your server, pull the changes:
```bash
cd ~/stillwaters-website
git pull
```

3. Restart your application:
```bash
# If using systemd:
sudo systemctl restart stillwaters

# Or if running directly:
pkill -f "go run main.go"
go run main.go &
```

4. Test the contact form on your website

---

## How the Solution Works

### Before (SMTP - ❌ Problem)
```
Your App → SMTP (port 587) → SES → Yahoo (DMARC check fails!)
FROM: c_glendenning@yahoo.com ← Yahoo blocks this!
```

### After (SES v2 API - ✅ Solution)
```
Your App → SES v2 API (IAM role auth) → Yahoo
FROM: noreply@stillwatersretreats.com ← You control this domain
Reply-To: c_glendenning@yahoo.com ← Easy to reply to customer
Body: Email: c_glendenning@yahoo.com ← Customer's email visible
```

### Benefits:
1. **Works reliably** - No DMARC failures
2. **Better deliverability** - Your domain builds reputation
3. **No SMTP credentials** - More secure with IAM role
4. **Easy replies** - Reply-To header makes it one-click
5. **Professional** - Emails come from your brand

---

## Troubleshooting

### "Email not verified"
- Make sure you've verified your domain or email in SES
- Wait for DNS propagation (up to 30 minutes)

### "Still in sandbox mode"
- You can only send TO verified emails in sandbox
- Request production access (Step 4)

### "Access denied" / "Credentials not found"
- Make sure IAM role is attached to your EC2 instance
- Restart your app after attaching the role

### "DMARC failure" still happening
- Make sure `SES_FROM_EMAIL` is using YOUR domain
- Don't use @yahoo.com, @gmail.com, etc. in FROM address

---

## Need to Update Your Code Later?

You can also remove the old gomail dependency if you want:
```bash
go mod tidy
```

This will clean up unused dependencies from `go.mod`.

