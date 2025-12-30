# Email Setup with Resend

WatchClub uses [Resend](https://resend.com) for sending transactional emails (login links). Resend offers a generous free tier of 3,000 emails per month.

## Quick Setup

### 1. Create a Resend Account

1. Go to [resend.com](https://resend.com) and sign up
2. Verify your email address

### 2. Add and Verify Your Domain

1. In the Resend dashboard, go to **Domains**
2. Click **Add Domain**
3. Enter your domain (e.g., `yourdomain.com`)
4. Add the DNS records shown to your domain's DNS settings
5. Wait for verification (usually takes a few minutes)

**Note**: For testing, you can skip domain verification and use Resend's test mode, but emails will only be sent to verified email addresses on your Resend account.

### 3. Create an API Key

1. In the Resend dashboard, go to **API Keys**
2. Click **Create API Key**
3. Give it a name (e.g., "WatchClub Production")
4. Select the **Sending access** permission
5. Copy the API key (you won't be able to see it again!)

### 4. Configure WatchClub

Run the server with email configuration:

```bash
./bin/watchclub server \
  --resend-api-key="re_..." \
  --resend-from="you@yourdomain.com" \
  --resend-from-name="WatchClub" \
  --base-url="https://yourdomain.com/"
```

Or use environment variables (recommended for production):

```bash
export RESEND_API_KEY="re_..."
export RESEND_FROM="you@yourdomain.com"
export RESEND_FROM_NAME="WatchClub"
export BASE_URL="https://yourdomain.com/"

./bin/watchclub server \
  --resend-api-key="$RESEND_API_KEY" \
  --resend-from="$RESEND_FROM" \
  --resend-from-name="$RESEND_FROM_NAME" \
  --base-url="$BASE_URL"
```

## Configuration Options

### Email Provider Flags

- `--resend-api-key`: Your Resend API key (get from resend.com dashboard)
- `--resend-from`: Email address to send from (must be from your verified domain)
- `--resend-from-name`: Display name for the sender (optional, defaults to email address)
- `--base-url`: Base URL for generating login links (e.g., `https://watchclub.example.com/`)

### Development Mode

- `--dev` or `-d`: Force development mode (logs emails to console instead of sending)

If no email provider is configured, WatchClub automatically falls back to development mode with a warning.

## Email Behavior

### Priority Order

1. If `--dev` flag is set → **Development mode** (console logging)
2. If `--resend-api-key` is provided → **Resend** (real emails)
3. Otherwise → **Development mode** (console logging with warning)

### Email Format

Login emails are sent with:
- **Subject**: "Log in to WatchClub"
- **From**: Your configured address (e.g., "WatchClub <you@yourdomain.com>")
- **Format**: Both HTML and plain text versions
- **Content**: Styled email with button and text link

## Testing

### Local Development

For local development, just run without email flags:

```bash
./bin/watchclub server
```

You'll see login links printed to the console:

```
📧 RECOVERY EMAIL (Development Mode) to=user@example.com userName=John recoveryLink=http://localhost:3000/#/login/abc123
```

### Testing with Resend

To test real email sending:

1. Use your Resend API key and verified domain
2. Send a test login email via the WatchClub UI
3. Check the Resend dashboard under **Logs** to see delivery status
4. Check your inbox!

## Troubleshooting

### Emails not sending

Check the server logs for errors:

```bash
./bin/watchclub server --resend-api-key="..." --resend-from="..." 2>&1 | grep -i email
```

Common issues:
- **Invalid API key**: Double-check you copied the key correctly
- **Domain not verified**: Verify your domain in the Resend dashboard
- **"From" address not on verified domain**: The `--resend-from` address must be on a domain you've verified

### Still seeing console logs instead of emails

Make sure:
- You're not using the `--dev` flag
- The `--resend-api-key` flag is set
- The `--resend-from` flag is set

## Production Deployment

### Kubernetes Example

Store sensitive values in secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: watchclub-email
type: Opaque
stringData:
  resend-api-key: "re_your_api_key_here"
  resend-from: "watchclub@yourdomain.com"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: watchclub
spec:
  template:
    spec:
      containers:
      - name: watchclub
        image: watchclub:latest
        command:
          - /app/watchclub
          - server
          - --resend-api-key=$(RESEND_API_KEY)
          - --resend-from=$(RESEND_FROM)
          - --resend-from-name=WatchClub
          - --base-url=https://watchclub.yourdomain.com/
        env:
        - name: RESEND_API_KEY
          valueFrom:
            secretKeyRef:
              name: watchclub-email
              key: resend-api-key
        - name: RESEND_FROM
          valueFrom:
            secretKeyRef:
              name: watchclub-email
              key: resend-from
```

## Free Tier Limits

Resend's free tier includes:
- **3,000 emails/month**
- **100 emails/day**
- **1 domain**
- **No credit card required**

This is more than enough for a personal WatchClub instance!
