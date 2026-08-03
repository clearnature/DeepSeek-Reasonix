# DeepSeek get-user-balance 文档

> 来源: https://api-docs.deepseek.com/zh-cn/api/get-user-balance

# 查询余额

GET 
## /user/balance

查询账号余额

## Responses​
- 200
OK, 返回用户余额详情

- application/json
- Schema
- Example (from schema)
- Example
Schema

is_available boolean当前账户是否有余额可供 API 调用

balance_infos

object[]

- Array [
currency stringPossible values: [`CNY`, `USD`]

货币，人民币或美元

total_balance string总的可用余额，包括赠金和充值余额

granted_balance string未过期的赠金余额

topped_up_balance string充值余额

- ]

```
{  "is_available": true,  "balance_infos": [    {      "currency": "CNY",      "total_balance": "110.00",      "granted_balance": "10.00",      "topped_up_balance": "100.00"    }  ]}
```

```
{  "is_available": true,  "balance_infos": [    {      "currency": "CNY",      "total_balance": "110.00",      "granted_balance": "10.00",      "topped_up_balance": "100.00"    }  ]}
```
Loading...