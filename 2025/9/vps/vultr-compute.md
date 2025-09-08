# Vultr

1. https://vultr.com/products/cloud-compute
2. click Create account
3. enter Email Address
4. enter Password
5. click Create free account
6. enter Your Name
7. enter Address
8. click Save this address
9. enter Card Number
10. enter expiration
11. enter CVV
12. click I Agree to the Terms of Service
13. click Link Credit Card
14. click Verify Your E-mail

## Deploy

1. click Cloud Compute
2. click intel Regular Performance
3. click IPv4 10 GB SSD
4. click Deploy Now
5. click Cloud Instance
6. click View Console
7. once you see Cloud-init finished, press enter
8. enter login `root`
9. Show the control bar
10. click Clipboard
11. paste password
12. click Paste
13. Hide the control bar

## Firewall

blocks HTTP by default, allow:

~~~
ufw allow http
ufw status verbose
~~~

https://digitalocean.com/community/tutorials/how-to-set-up-a-firewall-with-ufw-on-debian-10

## server

~~~
go/bin/go run umber-main/hello.go
~~~
