# avl-receiver

AVL = automatic vehicle locator

This tool hosts a tcp server to which supported AVL devices can connect and communicate.

### Instructions to run
```
docker build . -t avl-receiver
docker run -p 9000:9000 avl-receiver
```