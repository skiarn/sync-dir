# sync-dir

# sync-dir
**The server application is to be used to sync files from multiple remote servers using ssh.**

### How to get started ###

* Install go - https://golang.org/dl/
* git clone http://github.com/skiarn/sync-dir

Configuration
* linux/mac: ```export GOPATH=/project/path/sync-dir```
* windows: ```set GOPATH=/project/path/sync-dir```

### Build application ###
go install sync-dir

## Run application
Application is located in $GOPATH/bin.
* Example `./sync-dir -h=localhost -u=username -d=/home/username/server/logs`
* For help ```./sync-dir --help ```
```
Usage of ./sync-dir:
  -d="": Directory path on remote host server.
  -h="": Host to system to be synced with.
  -p=22: ssh port.
  -u="": User to be used when connecting to host.
```
