# AfterMail MailScript API Reference

MailScript is a powerful email filtering and automation scripting language based on Starlark (Python-like syntax). It allows you to create sophisticated email processing rules with a simple, familiar syntax.

## Overview

Scripts are executed for each incoming email message. You can define an `evaluate()` function as the main entry point, or write code at the root level. The script has access to email headers, body content, and metadata through built-in functions.

## Language Features

MailScript is built on **Starlark**, a Python-like language. It supports:

### Data Types
- **Strings**: `"hello"`, `'world'`, `"""multi-line"""`
- **Integers**: `42`, `-10`, `1000000`
- **Floats**: `3.14`, `1.5`, `0.001`
- **Booleans**: `True`, `False`
- **Lists** (arrays): `[1, 2, 3]`, `["a", "b", "c"]`
- **Dictionaries** (associative arrays): `{"key": "value", "score": 10}`
- **None**: `None`

### Lists (Arrays)
```python
# Create lists
spam_keywords = ["viagra", "cialis", "lottery"]
scores = [5.0, 7.5, 9.2]

# Access elements
first = spam_keywords[0]  # "viagra"
last = spam_keywords[-1]  # "lottery"

# Iterate
for keyword in spam_keywords:
    if search_body(keyword):
        quarantine()

# Check membership
if "viagra" in spam_keywords:
    fileinto("Spam")

# List operations
all_keywords = spam_keywords + ["winner", "free"]
length = len(spam_keywords)  # 3
```

### Dictionaries (Associative Arrays)
```python
# Create dictionaries
spam_scores = {
    "viagra": 10.0,
    "lottery": 8.5,
    "winner": 7.0
}

sender_actions = {
    "boss@company.com": "accept",
    "spam@bad.com": "drop",
    "client@vip.com": "priority"
}

# Access values
score = spam_scores["viagra"]  # 10.0

# Check if key exists
if "viagra" in spam_scores:
    log_entry("Spam keyword found")

# Iterate over keys
for keyword in spam_scores:
    log_entry(keyword + ": " + str(spam_scores[keyword]))

# Iterate over key-value pairs
for keyword, score in spam_scores.items():
    if search_body(keyword):
        add_header("X-Spam-Match", keyword)
        add_header("X-Spam-Score", str(score))
```

### Loops
```python
# For loop with list
trusted = ["alice@company.com", "bob@partner.org"]
sender = get_header("From")
for email in trusted:
    if email == sender:
        accept()
        return

# For loop with range
for i in range(5):
    log_entry("Processing step: " + str(i))

# For loop with dictionary
rules = {"spam": "drop", "ham": "accept"}
for category, action in rules.items():
    log_entry(category + " -> " + action)
```

**Note:** Starlark does **not** support `while` loops for safety reasons.

### Conditionals
```python
# If-elif-else
score = getspamscore()
if score > 9.0:
    quarantine()
elif score > 7.0:
    fileinto("Spam")
elif score > 5.0:
    add_header("X-Suspicious", "true")
else:
    accept()

# Comparison operators: ==, !=, <, >, <=, >=
# Logical operators: and, or, not
if score > 5.0 and getvirusstatus() == "clean":
    accept()
```

### Functions
```python
def check_spam(threshold):
    """Check if message is spam"""
    if getspamscore() > threshold:
        return True
    return False

def process_trusted_sender(sender):
    """Handle trusted senders"""
    skip_spam_check(sender)
    skip_malware_check(sender)
    accept()

def evaluate():
    sender = get_header("From")
    if regex_match(".*@trusted\\.com$", sender):
        process_trusted_sender(sender)
    elif check_spam(8.0):
        quarantine()
    else:
        accept()
```

### String Operations
```python
subject = get_header("Subject")

# Concatenation
msg = "Subject is: " + subject

# Case conversion
lower = subject.lower()
upper = subject.upper()

# Check contents
if "urgent" in subject.lower():
    add_header("X-Priority", "high")

# String methods
if subject.startswith("Re:"):
    log_entry("Reply detected")

if subject.endswith("?"):
    log_entry("Question detected")

# Format/replace
trimmed = subject.strip()
cleaned = subject.replace("[SPAM]", "")
```

### List Comprehensions
```python
# Create filtered lists
high_scores = [s for s in [5.0, 7.5, 9.2, 3.1] if s > 7.0]
# Result: [7.5, 9.2]

# Transform lists
uppercased = [word.upper() for word in ["spam", "ham", "eggs"]]
# Result: ["SPAM", "HAM", "EGGS"]
```

### Type Conversions
```python
# String to number (note: not directly supported in Starlark)
# Use int() and float() when available

# Number to string
score = getspamscore()
score_str = str(score)
add_header("X-Score", score_str)

# Boolean to string
is_spam = getspamscore() > 8.0
add_header("X-Is-Spam", str(is_spam))
```

## Message Actions

### accept()
Accept and deliver the message to the inbox.

```python
accept()
```

### discard()
Silently drop the message without delivering it.

```python
discard()
```

### fileinto(folder)
Move the message to a specific folder.

**Parameters:**
- `folder` (string): Name of the folder to move the message to

```python
fileinto("Spam")
fileinto("Archive/2024")
```

### quarantine()
Move the message to quarantine for manual review.

```python
quarantine()
```

### add_to_next_digest()
Add the message to the next digest email. Returns `True` if successful.

```python
if add_to_next_digest():
    print("Added to digest")
```

### drop()
Forcefully drop the message without processing.

```python
drop()
```

### bounce()
Bounce the message back to the sender.

```python
if getspamscore() > 9.0:
    bounce()
```

### reply_with_smtp_error(code)
Reply to the sender with a specific SMTP error code.

**Parameters:**
- `code` (int): SMTP error code (e.g., 550, 554, 421)

```python
reply_with_smtp_error(550)  # Mailbox unavailable
reply_with_smtp_error(554)  # Transaction failed
```

### reply_with_smtp_dsn(dsn)
Reply with an SMTP Delivery Status Notification.

**Parameters:**
- `dsn` (string): DSN string (e.g., "5.7.1", "5.1.1")

```python
reply_with_smtp_dsn("5.7.1")  # Delivery not authorized
reply_with_smtp_dsn("5.1.1")  # Bad destination mailbox address
```

## Header Operations

### get_header(name)
Get the value of an email header.

**Parameters:**
- `name` (string): Name of the header (e.g., "From", "Subject", "To")

**Returns:** String value of the header, or empty string if not found

```python
sender = get_header("From")
subject = get_header("Subject")
```

### add_header(name, value)
Add a custom header to the message.

**Parameters:**
- `name` (string): Header name
- `value` (string): Header value

```python
add_header("X-MailScript", "processed")
add_header("X-Spam-Score", "7.5")
```

## Content Search

### search_body(text)
Search for text in the message body.

**Parameters:**
- `text` (string): Text to search for (literal match)

**Returns:** Boolean (`True` if found, `False` otherwise)

```python
if search_body("viagra"):
    fileinto("Spam")
```

### regex_match(pattern, text)
Test if text matches a regular expression pattern.

**Parameters:**
- `pattern` (string): Regular expression pattern
- `text` (string): Text to test against the pattern

**Returns:** Boolean (`True` if matches, `False` otherwise)

```python
sender = get_header("From")
if regex_match(".*@trusted\\.com$", sender):
    accept()
```

## Message Metadata

### getmimetype()
Get the MIME type of the message.

**Returns:** String (e.g., "text/plain", "multipart/mixed", "application/pdf")

```python
mime = getmimetype()
if mime == "application/pdf":
    add_header("X-Has-PDF", "true")
```

### getspamscore()
Get the spam score of the message (0.0 to 10.0).

**Returns:** Float value representing spam probability

```python
score = getspamscore()
if score > 7.0:
    fileinto("Spam")
elif score > 5.0:
    add_header("X-Suspicious", "true")
```

### getvirusstatus()
Get the virus scan status of the message.

**Returns:** String: "clean", "infected", or "unknown"

```python
status = getvirusstatus()
if status == "infected":
    quarantine()
```

### get_recipient_did()
Get the Web3 DID (Decentralized Identifier) of the recipient.

**Returns:** String containing the DID

```python
did = get_recipient_did()
```

### body_size()
Get the size of the email body in bytes.

**Returns:** Integer (size in bytes)

```python
bs = body_size()
if bs > 1048576:  # 1MB = 1048576 bytes
    quarantine()
```

### header_size()
Get the size of the email headers in bytes.

**Returns:** Integer (size in bytes)

```python
hs = header_size()
if hs > 102400:  # 100KB
    log_entry("Large headers detected")
```

### num_envelope()
Get the number of envelope senders.

**Returns:** Integer (count of envelope senders)

```python
n = num_envelope()
if n < 10:
    accept()
else:
    log_entry("Suspicious: multiple envelope senders")
    quarantine()
```

## Routing Actions

### divert_to(email_address)
Redirect the message to a different email address.

**Parameters:**
- `email_address` (string): Email address to redirect to

```python
if regex_match(".*(invoice|payment).*", subject):
    divert_to("accounting@example.com")
```

### screen_to(email_address)
Send a copy of the message to another address for screening/review.

**Parameters:**
- `email_address` (string): Email address to send copy to

```python
if getspamscore() > 5.0:
    screen_to("admin@example.com")
```

### force_second_pass(mailserver)
Route the message to another mail server for additional processing.

**Parameters:**
- `mailserver` (string): Mail server hostname or address

```python
force_second_pass("priority-server.example.com")
```

## Security Controls

### skip_malware_check(sender)
Bypass malware scanning for messages from a specific sender.

**Parameters:**
- `sender` (string): Sender email address

```python
if regex_match(".*@trusted\\.com$", sender):
    skip_malware_check(sender)
```

### skip_spam_check(sender)
Bypass spam filtering for messages from a specific sender.

**Parameters:**
- `sender` (string): Sender email address

```python
skip_spam_check("newsletter@company.com")
```

### skip_whitelist_check(ip)
Bypass whitelist checking for a specific IP address.

**Parameters:**
- `ip` (string): IP address

```python
skip_whitelist_check("192.168.1.100")
```

## Data Loss Prevention (DLP)

### set_dlp(mode, target)
Set Data Loss Prevention policy for the message.

**Parameters:**
- `mode` (string): Policy mode (e.g., "always", "conditional")
- `target` (string): Target scope ("user", "domain", etc.)

```python
if search_body("confidential"):
    set_dlp("always", "domain")
```

### skip_dlp(mode, target)
Skip DLP checks for the message.

**Parameters:**
- `mode` (string): Policy mode
- `target` (string): Target scope

```python
skip_dlp("sometimes", "user")
```

## Response Actions

### auto_reply(text)
Send an automated reply to the sender.

**Parameters:**
- `text` (string): Reply message text

```python
auto_reply("Thank you for your email. We will respond within 24 hours.")
```

## Logging

### log_entry(message)
Create a log entry for debugging and auditing.

**Parameters:**
- `message` (string): Log message

```python
log_entry("Processing VIP email")
log_entry("Spam score: " + str(getspamscore()))
```

## Content Filtering

### get_content_filter()
Get the current content filter identifier.

**Returns:** String (content filter ID)

```python
cf = get_content_filter()
log_entry("Using filter: " + cf)
```

### get_content_filter_name()
Get the name of the current content filter.

**Returns:** String (content filter name)

```python
cf_name = get_content_filter_name()
add_header("X-Content-Filter", cf_name)
```

### get_content_filter_rules()
Get the current content filter rules as a dictionary.

**Returns:** Dictionary of filter rules

```python
cf_rules = get_content_filter_rules()
# Access individual rules from the dictionary
```

### set_content_filter_rules(rule)
Set or update content filter rules. Returns `True` if successful.

**Parameters:**
- `rule` (string): Rule definition

```python
if set_content_filter_rules("block_attachments=true"):
    log_entry("Filter rules updated")
```

## Instance Information

### get_instance()
Get the current processing instance identifier.

**Returns:** String (instance ID)

```python
instance = get_instance()
log_entry("Processing on instance: " + instance)
```

### get_instance_name()
Get the name of the current processing instance.

**Returns:** String (instance name)

```python
instance_name = get_instance_name()
add_header("X-Processed-By", instance_name)
```

## DNS and Network Functions

### get_sender_ip()
Get the IP address of the message sender.

**Returns:** String (IP address)

```python
sender_ip = get_sender_ip()
log_entry("Message from IP: " + sender_ip)
```

### get_sender_domain()
Get the domain of the message sender.

**Returns:** String (domain name)

```python
sender_domain = get_sender_domain()
add_header("X-Sender-Domain", sender_domain)
```

### dns_check(domain)
Check if a domain has valid DNS records.

**Parameters:**
- `domain` (string): Domain to check

**Returns:** Boolean (`True` if DNS resolves, `False` otherwise)

```python
sender_domain = get_sender_domain()
if not dns_check(sender_domain):
    log_entry("Invalid sender domain")
    reply_with_smtp_error(550)
    drop()
```

### dns_resolution(domain)
Resolve a domain to its IP address.

**Parameters:**
- `domain` (string): Domain to resolve

**Returns:** String (resolved IP address)

```python
domain = get_sender_domain()
ip = dns_resolution(domain)
log_entry("Domain " + domain + " resolves to " + ip)
```

### domain_resolution(sender, verify)
Resolve sender domain with optional verification.

**Parameters:**
- `sender` (string): Sender email address
- `verify` (bool): Whether to verify the resolution

**Returns:** Boolean (`True` if resolution succeeded, `False` otherwise)

```python
sender = get_header("From")
if not domain_resolution(sender, True):
    log_entry("Sender domain verification failed")
    quarantine()
```

### rbl_check(ip, rbl_server)
Check if an IP address is listed in a Real-time Blackhole List (RBL).

**Parameters:**
- `ip` (string): IP address to check
- `rbl_server` (string, optional): Specific RBL server to check against

**Returns:** Boolean (`True` if listed, `False` otherwise)

```python
sender_ip = get_sender_ip()
if rbl_check(sender_ip, "zen.spamhaus.org"):
    add_header("X-RBL-Listed", "spamhaus")
    quarantine()

# Check against default RBL servers
if rbl_check(sender_ip):
    drop()
```

### get_rbl_status()
Get comprehensive RBL status information.

**Returns:** Dictionary with keys:
- `listed` (bool): Whether IP is in RBL
- `rbl_name` (string): Name of RBL where listed

```python
rbl = get_rbl_status()
if rbl["listed"]:
    log_entry("Listed in RBL: " + rbl["rbl_name"])
    quarantine()
```

### valid_mx(domain)
Check if a domain has valid MX (Mail Exchange) records.

**Parameters:**
- `domain` (string): Domain to check

**Returns:** Boolean (`True` if valid MX records exist, `False` otherwise)

```python
sender_domain = get_sender_domain()
if not valid_mx(sender_domain):
    log_entry("No valid MX records for " + sender_domain)
    reply_with_smtp_error(550)
    bounce()
```

### get_mx_records(domain)
Get all MX records for a domain.

**Parameters:**
- `domain` (string): Domain to lookup

**Returns:** List of MX record strings

```python
sender_domain = get_sender_domain()
mx_records = get_mx_records(sender_domain)
for mx in mx_records:
    log_entry("MX: " + mx)
    # Check each MX record
```

### mx_in_rbl(domain, rbl_server)
Check if any of a domain's MX records are listed in an RBL.

**Parameters:**
- `domain` (string): Domain to check
- `rbl_server` (string, optional): Specific RBL server

**Returns:** Boolean (`True` if any MX is in RBL, `False` otherwise)

```python
sender_domain = get_sender_domain()
if mx_in_rbl(sender_domain, "zen.spamhaus.org"):
    log_entry("MX records in RBL")
    add_header("X-MX-Blacklisted", "true")
    quarantine()
```

### is_mx_ipv4(domain)
Check if a domain has IPv4 MX records.

**Parameters:**
- `domain` (string): Domain to check

**Returns:** Boolean (`True` if IPv4 MX exists, `False` otherwise)

```python
sender_domain = get_sender_domain()
if is_mx_ipv4(sender_domain):
    log_entry("Domain has IPv4 MX records")
```

### is_mx_ipv6(domain)
Check if a domain has IPv6 MX records.

**Parameters:**
- `domain` (string): Domain to check

**Returns:** Boolean (`True` if IPv6 MX exists, `False` otherwise)

```python
sender_domain = get_sender_domain()
if is_mx_ipv6(sender_domain):
    add_header("X-IPv6-Capable", "true")
```

## Received Headers Analysis

### check_received_header(level)
Get a specific Received header at the given level (0 = most recent, 1 = next hop, etc.).

**Parameters:**
- `level` (int): Header level (0-based, from top of stack)

**Returns:** String (Received header content)

```python
# Check the most recent hop
first_hop = check_received_header(0)
log_entry("First hop: " + first_hop)

# Check third hop to detect relay patterns
third_hop = check_received_header(2)
if "suspicious-relay.com" in third_hop:
    log_entry("Known spam relay detected")
    quarantine()

# Check deep in the chain for original sender
for level in range(5):
    header = check_received_header(level)
    if header != "":
        log_entry("Level " + str(level) + ": " + header)
```

### get_received_headers()
Get all Received headers as a list.

**Returns:** List of Received header strings (ordered from most recent to oldest)

```python
received = get_received_headers()
num_hops = len(received)

if num_hops > 10:
    log_entry("Suspicious: " + str(num_hops) + " mail hops")
    add_header("X-Hop-Count", str(num_hops))
    quarantine()

# Analyze each hop
for i, hop in enumerate(received):
    log_entry("Hop " + str(i) + ": " + hop)
    if "known-spammer.com" in hop:
        quarantine()
        return
```

## Complete Example

```python
def evaluate():
    # Get message details
    sender = get_header("From")
    subject = get_header("Subject")

    # Check for spam
    if getspamscore() > 8.0:
        add_header("X-Spam-Action", "quarantine")
        quarantine()
        return

    # Check for virus
    if getvirusstatus() == "infected":
        quarantine()
        return

    # Handle trusted senders
    if regex_match(".*@trusted\\.com$", sender):
        skip_spam_check(sender)
        skip_malware_check(sender)
        accept()
        return

    # Search for spam keywords in body
    if search_body("viagra") or search_body("cheap meds"):
        add_header("X-Spam-Keyword", "match")
        fileinto("Spam")
        return

    # Route invoices to accounting
    if regex_match(".*(invoice|payment).*", subject.lower()):
        divert_to("accounting@example.com")
        return

    # Default: accept
    accept()
```

## Best Practices

1. **Always have a default action**: End your script with `accept()` or another action
2. **Return early**: Use `return` after actions to avoid executing multiple conflicting actions
3. **Test patterns carefully**: Use regex101.com or similar tools to test regex patterns
4. **Be cautious with whitelist/skip functions**: Only use for truly trusted sources
5. **Log important actions**: Use `add_header()` to track what actions were taken
6. **Handle edge cases**: Check for empty headers or missing data
7. **Keep scripts simple**: Complex logic is harder to debug and maintain

## Security Considerations

- Scripts run in a sandboxed Starlark environment with no file system or network access
- Regular expressions are evaluated with timeouts to prevent ReDoS attacks
- Action combinations are validated before execution
- Malicious scripts cannot harm the system or access sensitive data

## Debugging

Add headers to track script execution:

```python
add_header("X-Debug-Step", "1-checked-spam")
if getspamscore() > 5.0:
    add_header("X-Debug-Step", "2-spam-detected")
    fileinto("Spam")
```

## Script Entry Points

1. **evaluate() function**: If defined, this is called as the main entry point
2. **Root-level code**: If no `evaluate()` function exists, root-level code is executed
3. **Default action**: If no actions are taken, `accept()` is called automatically
