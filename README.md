# blahcdn / proxy
A simple reverse proxy in go.

## Supports:
* HTTP/2
* Server Side Caching (only redis support for now, more to be added)
* HTTP/3 (Experimental)


## TODO: 
* mTLS support for better security
* Clustering, directing request to instance which is nearest to the origin
* Better caching strategy
* Support Websockets better

### Note: for HTTP/3 support in some modern browsers, you need to have a certificate for a publicly resolveable domain 
ex) example.com, not localhost

