<h1 align="center">sdns 📂  </h1>

<h5 align="center">A DNS server based on minimal static configuration</h5>

<br/>

[![Build Status](https://travis-ci.org/cirocosta/sdns.svg?branch=master)](https://travis-ci.org/cirocosta/sdns)


### Use cases

#### Make a domain always resolve to localhost

```
sudo sdns \
        --port 53 \
        --addr 127.0.0.11 \
        'domain=test.cirocosta.io,ip=192.168.0.103,ns=mynameserver.com' \
        'domain=*.cirocosta.io,ip=127.0.0.1,ip=10.0.0.10'
```


#### Retrieve information about each DNS request being performed

```
sudo sdns \
        --debug \               # logs the requests to 'stderr'
        --port 53 \             
        --addr 127.0.0.11 \
        --recursor 8.8.8.8
```
