# 创建ca证书用于访问https请求时签发证书
## 创建ca根证书及秘钥
> openssl req -new -x509 -newkey rsa:4096 -keyout ca_encrypt.key -out ca.crt -config openssl.cnf -days 3650 -subj '/C=CN/ST=GuangDong/L=ShenZhen/O=Demo/OU=IT/CN=demo.com/emailAddress=ca@demo.com' -passin pass:123456 -passout pass:123456
## 解密私钥
> openssl rsa -in ca_encrypt.key -out ca.key
## 解析证书
> openssl x509 -in ca.crt -noout -text
## 验证公钥和私钥是否匹配(输出的哈希值应该一致)
> openssl x509 -noout -modulus -in ca.crt | openssl md5  
> openssl rsa -noout -modulus -in ca.key | openssl md5