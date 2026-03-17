Here's what we need for mailscript:

We have:


accept();
discard();
fileinfo();
get_header();
regex_match();

What we could use:

search_body();
getmimetype();
getspamscore();
getvirusstatus();

think like:

text = viagra

body = search_body(text)
    if matched(body)
     ah = add_header("X-MailScript: textmatch
   }

msg = getmessage(message)

if(set(ah)) {
    Quarantine(msg);
  if(add_to_next_digest(msg)) {
        print "Added";
 }
}

you could also do things like:
    divert_to(email_address);
    screen_to(email_address);
    skip_malware_check(sender);
    skip_spam_check(sender);
    skip_whitelist_check(ip);
    force_second_pass(mailserver); # send to another server for processing
    set_dlp(always,user)
    set_dlp(always,domain)
    skip_skip_dlp(someties,domain) or user

we can do so much more with this.

 
