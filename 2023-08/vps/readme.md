# Blog

- <https://wikipedia.org/wiki/Virtual_private_server>
- https://duckdns.org

Install Go:

~~~
curl -L -O https://go.dev/dl/go1.20.3.linux-amd64.tar.gz
tar -x -f go1.20.3.linux-amd64.tar.gz
~~~

Install server:

~~~
curl -L -O https://github.com/USER/REPO/archive/refs/heads/main.tar.gz
tar -x -f main.tar.gz
~~~

I wanted to find at least 9 workable options, which are these:

1. Amazon Lightsail
2. Atlantic.Net
3. DigitalOcean
4. Google Compute Engine
5. kamatera
6. Linode Shared
7. lunanode
8. upcloud
9. Vultr

remove options with poor web paste, and we have:

1. DigitalOcean
   - 31 steps total
2. Amazon Lightsail
   - 42 steps total
3. Google Compute Engine
   - 65 steps total

So I will try DigitalOcean, but other options should be fine as well.

## Alibaba Cloud

https://alibabacloud.com

If you go here, it is not english:

https://console.aliyun.com

## Amazon EC2

https://aws.amazon.com/free/compute/lightsail-vs-ec2

## Amazon Lightsail (accepted 1)

30 steps for create plus 12 steps for Lightsail equals 42 steps.

https://aws.amazon.com/lightsail

## Atlantic.Net (accepted 2)

22 steps sign up plus 10 steps cloud equals 32 steps.

https://atlantic.net

## civo

At this point I am blocked:

> We are now performing some account checks. This shouldn't take long.
>
> You will receive an email when your account is fully active (don't forget to
> check your spam folder just in case).

https://civo.com/compute

## Clouding

Spanish:

https://clouding.io

## DigitalOcean (accepted 3)

23 steps sign up plus 8 steps droplet equals 31 steps.

https://digitalocean.com

## euserv

I think the site is broken:

1. click Payment
2. click I have read and accepted the terms
3. click Proceed checkout
4. click Payment
5. click Go to checkout

https://euserv.com

## fly.io

how to create a machine using the website

I just started with Fly, and I am on this page:

https://fly.io/dashboard/personal/machines

and I dont see any way to create a machine. All I see is this:

> No machines
>
> Get started by creating a new machine.
>
> [Go to docs](https://fly.io/docs/machines/working-with-machines)

which leads to a page called "Working with the Machines API". I dont want to
work with the API, I just want to click some buttons on the website and have it
spit out a machine. Is that possible?

https://github.com/superfly/docs/issues/663

## Google Cloud

Same as Google Compute Engine:

https://cloud.google.com/pricing/list

## Google Compute Engine (accepted 4)

25 steps account plus 40 steps compute equals 65 steps.

https://cloud.google.com/compute

## Heroku

> When it comes to running apps, containerization abstracts away the burden of
> managing hardware or virtual machines.

https://heroku.com/dynos

## hetzner

At this point ui.idenfy.com wants to use my camera. I am not accepting that, so
I click Block.

https://hetzner.com/cloud

## IBM

Could not place order. Problem authorizing the credit card.

https://ibm.com/cloud

## kamatera (accepted 5)

27 steps account plus 14 steps cloud equals 41 steps.

https://kamatera.com

## Linode Shared (accepted 6)

22 steps account plus 8 steps virtual machine equals 30 steps.

https://linode.com/products/shared

## lunanode (accepted 7)

24 steps account plus 6 steps virtual machine equals 30 steps.

https://lunanode.com/infrastructure

## Microsoft Azure

Bastion is in Updating state. Please refresh the page or try again later.

https://azure.microsoft.com/pricing/details/virtual-machines

## Oracle Cloud

Even if you allow all JavaScript, I still get an error:

> Error processing transaction
>
> We're unable to complete your sign up. Common sign up errors are due to: (a)
> Using prepaid cards. Oracle only accepts credit card and debit cards (b)
> Intentionally or unintentionally masking one's location or identity (c)
> Entering incomplete or inaccurate account details. Please try again if this
> applies to you. Otherwise, contact Oracle Customer Service.

https://oracle.com/cloud

## OVH cloud

Not valid for United States

https://ovhcloud.com

## scaleway

cheapest they have right now is €0.0137/hour, which is €0.3288/day, which is
€9.864/30 day, which is $10.84/30 day.

https://scaleway.com/en/virtual-instances

## upcloud (accepted 8)

22 steps account plus 11 steps deploy equals 33 steps.

https://upcloud.com/products/cloud-servers

## vps wala

You have to Choose a Domain. If you want to continue with IP only, you are
stuck at this point.

https://vpswala.org

## Vultr (accepted 9)

14 steps account plus 13 steps deploy plus one step firewall equals 28 steps.
This is less steps than other options, but pasting in the console is awkward.

https://vultr.com
