![Insecure](https://img.shields.io/badge/security_status-INSECURE-ff0066.svg) ![Unsupported](https://img.shields.io/badge/development_status-unsupported-red.svg) ![License BSDv2](https://img.shields.io/badge/license-BSDv2-brightgreen.svg)

github.com/appliedtrust/mytotp
====
MyTOTP is a simple Go client for the Time-Based One Time Password (TOTP) protocol.

**Works just like Google Authenticator for your Mac/Windows/Linux desktop.**

**_Warning:_ this software is still in development and probably not ready to trust with your most sensitive credentials.**

##Quickstart
1. Import a TOTP secret with a QR code or secret string.  
  1a. Save the QR code to a local file, then run `mytotp qr <filename> ; rm <filename>`  
  *OR*  
  1b. Find the "secret string", then run `mytotp add <name> <secret>`  
      *You might have to click on "manual entry" in AWS or Google to see this string.*
2. Run `mtotp -w` to see your code.
3. Secrets are stored encrypted in `~/.totp` -- **don't** forget your passphrase!

##Usage

```unix
mytotp: TOTP client for the command line

Usage:
  mytotp [options]
  mytotp add <name> <secret>
  mytotp qr <filename>
  mytotp dump
  mytotp import <filename>
  mytotp -h --help
  mytotp --version

Options:
  -s, --secrets <file>        TOTP secrets file - KEEP PRIVATE [default: ~/.totp]. 
  -c, --continuous            Print codes continuously.
  -w, --web                   Run a simple web interface.
  -l, --listen <ip:port>      Local IP and port for web interface [default: localhost:8000].
  -h, --help                  Show this screen.
  --version                   Show version.

Optionally set the MYTOTP_PASSPHRASE environment variable to avoid the initial passphrase prompt.
```


**DANGER**: Running this tool on the same workstation you login from probably violates NIST-800-63 best-practices for two-factor.  Similarly, using the `mytotp dump` feature puts your secrets at risk of compromise.

