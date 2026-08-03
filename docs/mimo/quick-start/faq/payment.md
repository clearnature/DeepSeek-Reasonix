# Payment

### How to recharge on the Open Platform? 

- **Pay-as-you-go API:** Go to the [ Account Balance ](https://platform.xiaomimimo.com/#/console/balance) page to top up. The platform provides three top-up methods for domestic users: Xiaomi Pay, Alipay, and WeChat Pay; for overseas users, it provides  **commonly used top-up methods such as Apple Pay, Google Pay, credit/debit cards, etc.**  Top-ups generally arrive in real-time, and you can check the balance in the [ Account Balance ](https://platform.xiaomimimo.com/#/console/balance), and view the cumulative top-up amount and top-up records on the [ Top-up Details ](https://platform.xiaomimimo.com/#/console/recharge)  page. 

- **Token Plan:** The plan does not currently support deduction using account balance or bonus, and you need to go to [Subscribe to Token Plan](https://platform.xiaomimimo.com/token-plan) to purchase separately.

<br></br>

### Does purchasing a Token Plan package count towards cumulative recharge?

Subscriptions are not counted, and orders for subscription packages are not included in cumulative recharge. 

<br></br>

### What payment methods are supported? 

Domestically in China, it supports WeChat Pay, Alipay, and Xiaomi Pay, while overseas, it uses the Waffo payment gateway for payment (settled in US dollars $).

<br></br>

### Is there a time limit for payment?

The effective payment duration is subject to the display on the page. After the timeout, the order will automatically close, and you need to place a new order. 

<br></br>

### How to set up balance alerts?

On the [Account Balance](https://platform.xiaomimimo.com/#/console/balance) page, you can enable the balance alert setting. Once enabled, when the account balance falls below the alert threshold, we will send a notification to your registered mobile number/email. Please check it carefully.

<br></br>

### Does it support refund applications?

- **Pay-as-you-go API:** Account balance supports full refund. If you have a refund request, you can open the contact pop-up window via the "Apply for Refund" button in the upper right corner of [Recharge Details](https://platform.xiaomimimo.com/#/console/recharge) , select the "Refund" option, and state the reason to initiate a refund request. The account balance will be refunded back to the original payment method after the review is approved (the consumed amount, the invoiced amount, and the platform-gifted amount cannot be refunded). After the refund request is accepted, you will no longer be able to continue calling the model service and will not be able to continue recharging. Billing may be delayed, and the refund amount shall be subject to the actual amount received. Refunds are generally processed and returned to the original payment method within 3-5 working days.

- **Token Plan:** Once a package is paid, it cannot be refunded.

<br></br>

### How to issue an invoice?

- **Chinese User:**

Visit the [Invoice](https://platform.xiaomimimo.com/#/console/invoice) page, select the successfully recharged order, and issue an electronic invoice. Both individual and corporate invoices can be issued. Fill in your email or mobile phone number, and the invoice will be sent via email/sms upon completion of issuance.

Note:
- The amount eligible for invoicing is the actual payment amount. Platform coupons, discounts, or amounts gifted by the platform cannot be invoiced. Refunded amounts cannot be invoiced. 
- According to regulations, invoices issued in the name of individuals can only be issued as digital electronic ordinary invoices, while invoices issued in the name of enterprises can be issued as digital electronic ordinary invoices or digital electronic special invoices.
- We generally issue invoices within 48 hours after receiving the application, and delays may occur in case of special circumstances. 
- Invoices support red stamping. If the invoice has been deducted or recorded, you need to log in to the Electronic Tax Bureau and confirm the information (red stamping confirmation form) within 72 hours to successfully complete the red stamping. After red stamping, the original order can be reissued with an invoice. 
- The invoicing entity is: Beijing Xiaomi Mobile Software Co., Ltd. 

- **Overseas Users:** 

Each recharge order will automatically generate an invoice. When you complete a recharge, you can view the invoice on the order page. You can also enter the [Recharge Details](https://platform.xiaomimimo.com/#/console/recharge)  page to download historical invoices. 

<br></br>

### Can I still call the API if my balance is insufficient? 

Before the billing system goes live, models can be called for inference services normally when the balance is 0. 

After the billing system goes live, due to a certain time delay, the balance may be ≤ 0. Once the balance becomes negative, the model inference service can no longer be used, and the next recharge order will first deduct the overdue amount. 

<br></br>

###  Will there still be charges after the API Key is deleted?

If the API Key is deleted, it will no longer be able to call the interface, and no charges will be incurred. The historical consumption records of this API Key can still be queried in [Billing](https://platform.xiaomimimo.com/#/console/usage).
