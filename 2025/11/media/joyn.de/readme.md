# register

1. joyn.de
2. email
   1. mailsac.com fail
   2. gmail.com pass
   3. mail.tm?
3. password
   - Das Passwort muss mindestens 1 Klein- und Großbuchstaben sowie eine Zahl
   enthalten (The password must contain at least one lowercase and one uppercase
   letter, as well as one number)
4. Männlich (male)
5. day
6. month
7. year
8. save
9. PLUS
10. try now for free
11. enable JavaScript
12. first name
13. surname
14. Zahlungsmethode wählen (select payment method)
15. credit card
16. Karteninhaber (cardholder)
17. card number
18. CVC
19. month
20. year
21. I expressly agree to the execution of the contract
22. ordering the payment

## prompt

a service offers these payment methods

1. visa
2. mastercard
3. american express
4. paypal
5. amazon pay

however I get this when using Amazon Pay:

CheckoutSession cannot be completed: the billing address country US is not
included in the allowed whitelist. Allowed countries: DE

and similar errors with the other methods. how can I get one of these payment
methods? also do not offer guesses. also do not offer vague ideas but concrete
options or steps. also the service does not ask for the address, it must find the
address that is linked to the card, so dont suggest to enter a german address
when making a purchase because that is not an option. also dont offer a service
like Wise unless you are certain they offer german cards. also dont offer a
service unless you are certain I can get it as a US citizen

## 1 visa US

~~~
POST /api/v1/CustomerSelfService/signup/pay HTTP/2
Host: p7.billwerk.com

{
  "paymentData": {
    "bearer": {
      "token": "tok_1SRN66EPPEU9tzM9FQskkgIJ"
    }
  }
}

HTTP/2 200 OK

{
  "Error": {
    "Code": "BearerInvalid",
    "Message": "Payment failed!",
    "Details": "Your card was declined."
  }
}
~~~

## 2 mastercard US

~~~
POST /api/v1/CustomerSelfService/signup/pay HTTP/2
Host: p7.billwerk.com

{
  "paymentData": {
    "bearer": {
      "token": "tok_1SRN66EPPEU9tzM9FQskkgIJ"
    }
  }
}

HTTP/2 200 OK

{
  "Error": {
    "Code": "BearerInvalid",
    "Message": "Payment failed!",
    "Details": "Your card was declined."
  }
}
~~~

## 3 american express US

~~~
POST /api/v1/CustomerSelfService/signup/pay HTTP/2
Host: p7.billwerk.com

{
  "paymentData": {
    "bearer": {
      "token": "tok_1SRN66EPPEU9tzM9FQskkgIJ"
    }
  }
}

HTTP/2 200 OK

{
  "Error": {
    "Code": "BearerInvalid",
    "Message": "Payment failed!",
    "Details": "Your card was declined."
  }
}
~~~

## 4 paypal US

~~~
POST https://p7.billwerk.com/api/v1/CustomerSelfService/Finalize HTTP/2.0

{
  "returnUrl": "https://www.joyn.de/abo/v2/willkommen?productVariantId=de-plus-web-monthly&interactivePayment=true&pactasTransactionId=690febaa2a093d3441958ec6&secret=Qye5089O39XJWIIYe9sdzQ&trigger=Payment&language=de-DE&token=EC-9E522846T9953502C"
}

HTTP/2.0 422 

{
  "Error": {
    "Code": "",
    "Message": "The country in the address field does not match country in payment",
    "Details": ""
  }
}
~~~

## 5 amazon.de US card

~~~json
POST https://p7.billwerk.com/api/v1/CustomerSelfService/Finalize HTTP/2.0

{
  "returnUrl": "https://www.joyn.de/abo/v2/willkommen?productVariantId=de-plus-web-monthly&interactivePayment=true&pactasTransactionId=69100f1a8d6ed9975022459e&secret=0sYLF0qPunj8F3s5FoVj5g&trigger=Payment&language=de-DE&amazonCheckoutSessionId=bfd7aff5-c6db-4f6c-805f-51135e87e5ba"
}

HTTP/2.0 200 

{
  "Error": {
    "Code": "InvalidCountry",
    "Message": "Payment failed!",
    "Details": "CheckoutSession cannot be completed: the billing address country US is not included in the allowed whitelist. Allowed countries: DE"
  }
}
~~~

## 6 VizoVcc 

https://vizovcc.com/card/details/mastercard-de/23

- 532454
- DE
- $150

## 7 yellowdotpay.net

➤ Issued from: Trusted Turkish Bank
➤ Bank Issued: Verified Argentine Bank

https://yellowdotpay.net/blog/virtual-credit-card-germany
