# AfterMail MailScript Examples
# This file demonstrates all available mailscript functions

def evaluate():
    """
    Main evaluation function that processes incoming email
    """

    # Example 1: Search body for spam keywords and quarantine
    if search_body("viagra") or search_body("cheap meds"):
        add_header("X-MailScript", "spam-keyword-match")
        quarantine()
        add_to_next_digest()
        return

    # Example 2: Check spam score and handle accordingly
    spam_score = getspamscore()
    if spam_score > 7.0:
        add_header("X-Spam-Score", str(spam_score))
        fileinto("Spam")
        return
    elif spam_score > 5.0:
        add_header("X-Suspicious", "high-spam-score")
        screen_to("admin@example.com")

    # Example 3: Check virus status
    virus_status = getvirusstatus()
    if virus_status == "infected":
        add_header("X-Virus-Detected", virus_status)
        quarantine()
        return

    # Example 4: Trusted sender - skip checks
    sender = get_header("From")
    if regex_match(".*@trusted-domain\\.com$", sender):
        skip_spam_check(sender)
        skip_malware_check(sender)
        accept()
        return

    # Example 5: Check MIME type and handle attachments
    mime_type = getmimetype()
    if mime_type == "application/pdf":
        add_header("X-PDF-Attachment", "true")

    # Example 6: VIP handling with auto-reply
    if regex_match(".*@vip-client\\.com$", sender):
        auto_reply("Thank you for your email. We will respond within 24 hours.")
        fileinto("VIP")
        return

    # Example 7: Compliance and DLP
    subject = get_header("Subject")
    if regex_match(".*(confidential|secret|proprietary).*", subject.lower()):
        set_dlp("always", "domain")
        add_header("X-DLP-Policy", "confidential")

    # Example 8: Route certain emails for additional processing
    if regex_match(".*URGENT.*", subject):
        force_second_pass("priority-mailserver.example.com")

    # Example 9: Divert invoices to accounting
    if regex_match(".*(invoice|payment|receipt).*", subject.lower()):
        divert_to("accounting@example.com")
        return

    # Example 10: Whitelist check bypass for known IPs
    # Note: In real implementation, you'd get sender IP from headers
    sender_ip = get_header("X-Originating-IP")
    if sender_ip == "192.168.1.100":
        skip_whitelist_check(sender_ip)

    # Default action: accept the message
    accept()


# Example 2: Simple spam filter
def spam_filter():
    """Simple spam detection based on body content"""
    spam_keywords = ["viagra", "cialis", "weight loss", "lottery winner"]

    for keyword in spam_keywords:
        if search_body(keyword):
            add_header("X-Spam-Keyword", keyword)
            fileinto("Spam")
            return

    accept()


# Example 3: Vacation auto-responder
def vacation_responder():
    """Auto-reply for vacation/out-of-office"""
    sender = get_header("From")

    # Don't auto-reply to mailing lists or automated emails
    if regex_match(".*noreply.*", sender) or regex_match(".*mailer-daemon.*", sender):
        accept()
        return

    auto_reply("""
    Thank you for your email. I am currently out of the office and will return on Monday.
    For urgent matters, please contact support@example.com.

    Best regards
    """)

    accept()


# Example 4: Advanced security filter
def security_filter():
    """Comprehensive security filtering"""

    # Check virus status
    if getvirusstatus() == "infected":
        add_header("X-Security-Action", "virus-quarantine")
        quarantine()
        return

    # Check spam score
    if getspamscore() > 8.0:
        add_header("X-Security-Action", "high-spam-score")
        quarantine()
        return

    # Check for suspicious attachments
    mime = getmimetype()
    dangerous_types = ["application/x-msdownload", "application/x-executable"]
    if mime in dangerous_types:
        add_header("X-Security-Action", "dangerous-attachment")
        screen_to("security@example.com")
        quarantine()
        return

    # Check body for phishing indicators
    phishing_patterns = ["verify your account", "click here immediately", "suspended account"]
    for pattern in phishing_patterns:
        if search_body(pattern):
            add_header("X-Security-Action", "phishing-suspected")
            add_header("X-Phishing-Pattern", pattern)
            quarantine()
            return

    accept()


# Example 5: Size-based filtering
def size_filter():
    """Filter messages based on size constraints"""
    log_entry("Starting size-based filtering")

    # Check body size
    bs = body_size()
    if bs > 10485760:  # 10MB
        log_entry("Message body exceeds 10MB: " + str(bs))
        reply_with_smtp_error(552)  # Message size exceeds fixed maximum
        drop()
        return
    elif bs > 5242880:  # 5MB
        log_entry("Large message detected: " + str(bs))
        add_header("X-Large-Message", "true")
        fileinto("Large Messages")
        return

    # Check header size
    hs = header_size()
    if hs > 102400:  # 100KB
        log_entry("Suspicious large headers: " + str(hs))
        add_header("X-Large-Headers", "true")
        quarantine()
        return

    accept()


# Example 6: Envelope sender filtering
def envelope_filter():
    """Filter based on envelope sender count"""
    log_entry("Checking envelope senders")

    n = num_envelope()
    log_entry("Envelope sender count: " + str(n))

    if n == 0:
        log_entry("No envelope senders - suspicious")
        reply_with_smtp_dsn("5.7.1")  # Delivery not authorized
        drop()
        return
    elif n > 10:
        log_entry("Multiple envelope senders detected")
        add_header("X-Multi-Sender", str(n))
        quarantine()
        return
    elif n < 10:
        # Normal processing
        accept()


# Example 7: Instance-aware processing
def instance_router():
    """Route based on processing instance"""
    instance = get_instance()
    instance_name = get_instance_name()

    log_entry("Processing on instance: " + instance_name)
    add_header("X-Processed-Instance", instance_name)

    # Route VIP messages to priority instance
    sender = get_header("From")
    if regex_match(".*@vip\\.com$", sender):
        if instance_name != "priority-instance":
            force_second_pass("priority-mailserver.example.com")
            return

    accept()


# Example 8: Content filter management
def content_filter_example():
    """Demonstrate content filter functions"""
    cf_name = get_content_filter_name()
    log_entry("Active content filter: " + cf_name)

    # Get current filter rules
    cf_rules = get_content_filter_rules()

    # Check if strict filtering is needed
    subject = get_header("Subject")
    if regex_match(".*CONFIDENTIAL.*", subject):
        log_entry("Applying strict content filtering")
        set_content_filter_rules("strict_mode=true,scan_attachments=true")
        add_header("X-Filter-Mode", "strict")

    accept()


# Example 9: Comprehensive bounce handler
def bounce_handler():
    """Handle bounces and SMTP errors appropriately"""

    # Check if sender is valid
    sender = get_header("From")
    if sender == "" or regex_match(".*noreply.*", sender):
        log_entry("Invalid or no-reply sender")
        drop()
        return

    # Check spam score - bounce high spam
    if getspamscore() > 9.5:
        log_entry("Extreme spam score - bouncing")
        reply_with_smtp_error(554)  # Transaction failed
        bounce()
        return

    # Check body size limits
    bs = body_size()
    if bs > 52428800:  # 50MB
        log_entry("Message too large")
        reply_with_smtp_error(552)  # Message exceeds fixed maximum
        bounce()
        return

    # Check for invalid recipients
    n = num_envelope()
    if n > 100:
        log_entry("Too many recipients")
        reply_with_smtp_dsn("5.7.1")  # Delivery not authorized
        drop()
        return

    accept()


# Example 10: Complete production filter with all features
def production_filter():
    """Production-ready filter using all available functions"""

    # Get instance info for logging
    instance_name = get_instance_name()
    log_entry("[" + instance_name + "] Starting message processing")

    # Get message metadata
    sender = get_header("From")
    subject = get_header("Subject")
    bs = body_size()
    hs = header_size()
    n = num_envelope()

    # Log message details
    log_entry("From: " + sender + ", Subject: " + subject)
    log_entry("Body size: " + str(bs) + ", Header size: " + str(hs))

    # Size checks
    if bs > 10485760:  # 10MB
        log_entry("Message too large")
        reply_with_smtp_error(552)
        drop()
        return

    if hs > 102400:  # 100KB
        log_entry("Suspicious large headers")
        quarantine()
        return

    # Envelope checks
    if n == 0 or n > 50:
        log_entry("Invalid envelope sender count: " + str(n))
        reply_with_smtp_dsn("5.7.1")
        drop()
        return

    # Security checks
    virus = getvirusstatus()
    if virus == "infected":
        log_entry("Virus detected")
        quarantine()
        return

    spam = getspamscore()
    if spam > 8.0:
        log_entry("High spam score: " + str(spam))
        if spam > 9.5:
            bounce()
        else:
            fileinto("Spam")
        return

    # Content filtering
    cf_name = get_content_filter_name()
    add_header("X-Content-Filter", cf_name)

    # VIP handling
    if regex_match(".*@vip\\.com$", sender):
        log_entry("VIP sender detected")
        skip_spam_check(sender)
        skip_malware_check(sender)
        fileinto("VIP")
        auto_reply("Thank you. Your message will be prioritized.")
        return

    # Business logic routing
    if regex_match(".*(invoice|payment).*", subject.lower()):
        log_entry("Invoice/payment email - routing to accounting")
        divert_to("accounting@example.com")
        return

    if search_body("confidential") or search_body("proprietary"):
        log_entry("Confidential content detected")
        set_dlp("always", "domain")
        add_header("X-DLP-Applied", "true")

    # Default action
    log_entry("Message accepted")
    accept()


# Example 11: DNS and RBL checking
def dns_rbl_filter():
    """Comprehensive DNS and RBL filtering"""
    log_entry("Starting DNS/RBL checks")

    sender = get_header("From")
    sender_ip = get_sender_ip()
    sender_domain = get_sender_domain()

    # Check if sender domain has valid DNS
    if not dns_check(sender_domain):
        log_entry("Sender domain has no valid DNS: " + sender_domain)
        reply_with_smtp_error(550)  # Mailbox unavailable
        drop()
        return

    # Verify DNS resolution matches
    if not domain_resolution(sender, True):
        log_entry("Domain resolution verification failed")
        add_header("X-DNS-Verify", "failed")
        quarantine()
        return

    # Check if sender IP is in RBL
    if rbl_check(sender_ip, "zen.spamhaus.org"):
        log_entry("Sender IP in Spamhaus: " + sender_ip)
        add_header("X-RBL-Listed", "spamhaus")
        quarantine()
        return

    # Get detailed RBL status
    rbl = get_rbl_status()
    if rbl["listed"]:
        log_entry("Listed in RBL: " + rbl["rbl_name"])
        add_header("X-RBL-Name", rbl["rbl_name"])
        drop()
        return

    # Check if domain has valid MX records
    if not valid_mx(sender_domain):
        log_entry("No valid MX records for: " + sender_domain)
        reply_with_smtp_dsn("5.1.1")  # Bad destination mailbox
        bounce()
        return

    # Check if MX records are in RBL
    if mx_in_rbl(sender_domain, "zen.spamhaus.org"):
        log_entry("MX records blacklisted: " + sender_domain)
        add_header("X-MX-Blacklisted", "true")
        quarantine()
        return

    log_entry("DNS/RBL checks passed")
    accept()


# Example 12: MX record analysis
def mx_analysis():
    """Analyze MX records for sender validation"""
    sender_domain = get_sender_domain()

    log_entry("Analyzing MX records for: " + sender_domain)

    # Get all MX records
    mx_records = get_mx_records(sender_domain)
    mx_count = len(mx_records)

    log_entry("Found " + str(mx_count) + " MX records")

    if mx_count == 0:
        log_entry("No MX records - rejecting")
        reply_with_smtp_error(550)
        drop()
        return

    # Check each MX record
    for mx in mx_records:
        log_entry("MX Record: " + mx)

        # Check for suspicious patterns
        if "dynamic" in mx or "dhcp" in mx:
            log_entry("Suspicious MX record: " + mx)
            quarantine()
            return

    # Check IP version support
    has_ipv4 = is_mx_ipv4(sender_domain)
    has_ipv6 = is_mx_ipv6(sender_domain)

    if has_ipv4:
        add_header("X-MX-IPv4", "true")
    if has_ipv6:
        add_header("X-MX-IPv6", "true")

    if not has_ipv4 and not has_ipv6:
        log_entry("No valid MX IP addresses")
        quarantine()
        return

    accept()


# Example 13: Received headers analysis
def received_analysis():
    """Analyze Received headers for spam relay detection"""
    log_entry("Analyzing Received headers")

    # Get all received headers
    received = get_received_headers()
    hop_count = len(received)

    log_entry("Total hops: " + str(hop_count))
    add_header("X-Hop-Count", str(hop_count))

    # Too many hops might indicate spam relay
    if hop_count > 15:
        log_entry("Excessive mail hops: " + str(hop_count))
        add_header("X-Relay-Suspicious", "excessive-hops")
        quarantine()
        return

    # Analyze each hop
    suspicious_relays = [
        "spam-relay.com",
        "open-relay.net",
        "compromised-server.org"
    ]

    for level in range(min(5, hop_count)):
        header = check_received_header(level)
        log_entry("Hop " + str(level) + ": " + header[:50] + "...")

        # Check for known spam relays
        for relay in suspicious_relays:
            if relay in header:
                log_entry("Known spam relay at hop " + str(level) + ": " + relay)
                add_header("X-Spam-Relay", relay)
                add_header("X-Spam-Relay-Level", str(level))
                quarantine()
                return

        # Check for suspicious patterns in early hops
        if level < 3:
            if "dynamic" in header or "dhcp" in header:
                log_entry("Dynamic IP in early hop: " + str(level))
                add_header("X-Dynamic-Hop", str(level))
                # Increase spam score but don't auto-quarantine
                if getspamscore() > 5.0:
                    quarantine()
                    return

    accept()


# Example 14: Complete network validation
def network_validator():
    """Comprehensive network-based validation"""
    sender = get_header("From")
    sender_ip = get_sender_ip()
    sender_domain = get_sender_domain()

    log_entry("=== Network Validation Start ===")
    log_entry("Sender: " + sender)
    log_entry("IP: " + sender_ip)
    log_entry("Domain: " + sender_domain)

    # Step 1: DNS validation
    if not dns_check(sender_domain):
        log_entry("[FAIL] DNS check failed")
        reply_with_smtp_error(550)
        drop()
        return
    log_entry("[PASS] DNS check")

    # Step 2: Domain resolution
    resolved_ip = dns_resolution(sender_domain)
    log_entry("Resolved IP: " + resolved_ip)

    # Step 3: MX validation
    if not valid_mx(sender_domain):
        log_entry("[FAIL] No valid MX records")
        reply_with_smtp_error(550)
        bounce()
        return
    log_entry("[PASS] MX validation")

    # Step 4: RBL checks
    if rbl_check(sender_ip):
        log_entry("[FAIL] IP in RBL")
        rbl = get_rbl_status()
        add_header("X-RBL-Listed", rbl["rbl_name"])
        quarantine()
        return
    log_entry("[PASS] RBL check")

    # Step 5: MX RBL check
    if mx_in_rbl(sender_domain):
        log_entry("[FAIL] MX in RBL")
        quarantine()
        return
    log_entry("[PASS] MX RBL check")

    # Step 6: Received headers analysis
    received = get_received_headers()
    hop_count = len(received)
    if hop_count > 10:
        log_entry("[WARN] High hop count: " + str(hop_count))
        add_header("X-High-Hop-Count", str(hop_count))

    # Check for relay patterns
    for level in range(min(3, hop_count)):
        hop = check_received_header(level)
        if "untrusted" in hop or "spam" in hop:
            log_entry("[FAIL] Suspicious relay at level " + str(level))
            quarantine()
            return

    log_entry("[PASS] Received headers check")

    # Step 7: IPv4/IPv6 validation
    if not is_mx_ipv4(sender_domain) and not is_mx_ipv6(sender_domain):
        log_entry("[FAIL] No IP connectivity")
        quarantine()
        return
    log_entry("[PASS] IP connectivity check")

    log_entry("=== Network Validation Complete: PASSED ===")
    add_header("X-Network-Validated", "true")
    accept()


# Example 15: Whitelist based on network criteria
def network_whitelist():
    """Whitelist trusted senders based on network validation"""
    sender_domain = get_sender_domain()
    sender_ip = get_sender_ip()

    # Trusted domains with verified network setup
    trusted_domains = [
        "google.com",
        "microsoft.com",
        "salesforce.com"
    ]

    if sender_domain in trusted_domains:
        # Extra validation for trusted domains
        if valid_mx(sender_domain) and not rbl_check(sender_ip):
            log_entry("Trusted domain verified: " + sender_domain)
            skip_spam_check(sender_domain)
            skip_malware_check(sender_domain)
            skip_dlp("sometimes", "domain")
            accept()
            return

    # Trusted IP ranges (example: company offices)
    if sender_ip.startswith("192.168.1.") or sender_ip.startswith("10.0."):
        log_entry("Internal IP detected: " + sender_ip)
        skip_whitelist_check(sender_ip)
        accept()
        return

    # Default processing
    accept()
