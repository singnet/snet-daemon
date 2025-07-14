# Rate Limiting

```snet-daemon``` uses the [Token Bucket](https://en.wikipedia.org/wiki/Token_bucket) Algorithm
 and the [rate](https://godoc.org/golang.org/x/time/rate) library. 
                  

###rate limiting configuration 
   * **rate_limit_per_minute** (optional; default: `Infinity`) 
   Defines the rate r at which the bucket is filled with tokesn per minute.
   By default this parameter is set to infinity, effectively having no rate limiting.

   * **burst_size** (optional; default: `Infinity`) -
   Defines a "token bucket" of size b , with a maximum burst size of b events.
   The Burst size is ignored when the rate limit is infinity.
   Please note that the Burst size is ignored when the rate limit is infinity.
 
### Configuration in JSON format
The below is an example on how rate limiting could be defined
```json
  {
    "burst_size": 80000,
    "rate_limit_per_minute": 50000
  }
```
### Usage details
For example
 if rate_limit_per_minute=1 and burst_size=1 => one request is served per minute
 if rate_limit_per_minute=0.5 and burst_size=1 =>  one request is served in every 2 minutes 