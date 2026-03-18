# AfterMail Website

Landing page for https://aftermail.app

## Features

- **Standalone MailScript** - Highlighted as a separate product that can be integrated into external systems
- **Mail Filtering Technology** - Showcases how MailScript brings enterprise-grade filtering to any infrastructure
- **Use Cases** - Email service providers, DevOps, security research, education, migrations
- **Getting Started** - Quick start guides for both AfterMail and MailScript

## Deployment

```bash
# Deploy to production
cd website
./deploy.sh
```

The site is served via nginx in a Docker container on apps.afterdarksys.com with Traefik handling SSL/TLS certificates.

## Local Development

Simply open `index.html` in a browser to preview changes.

## Key Messaging

1. ✅ MailScript can run standalone without AfterMail
2. ✅ Brings AfterMail's filtering technology to external systems
3. ✅ Perfect for integration into existing email infrastructure
4. ✅ Multiple deployment options: standalone binary, daemon integration, JSON automation
