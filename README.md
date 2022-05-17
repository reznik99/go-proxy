

## Commands:
<br/>

### Generate CA *Cert* & *Key*.
> openssl genrsa -out ca.key 2048

> openssl req -new -key ca.key -out ca.csr

> openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -out ca.pem
<br/>

### Generate Leaf *Key* and *CSR*.
> openssl genrsa -out intercept.key 2048

> openssl req -new -key intercept.key -out intercept.csr
<br/>

### Generate Leaf Certificate for Webserver using *CA*, *CSR* and *extensions* file.
> openssl x509 -req -in intercept.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out intercept.crt -days 365 -sha256 -extfile intercept.ext