#!/bin/bash
set -e

echo "🚀 Deploying aftermail.app website..."

# Copy files to server
echo "📦 Uploading files..."
scp index.html root@apps.afterdarksys.com:/root/aftermail.app/

echo "🔄 Restarting container..."
ssh root@apps.afterdarksys.com 'cd /root/aftermail.app && docker-compose restart'

echo "✅ Deployment complete!"
echo "🌐 Visit: https://aftermail.app"
