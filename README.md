# GoBlobStore
A multi-tenant proxy service for storing binary data in various storage systems with a simple HTTP interface.

features
- multi tenant binary store with strong separation of data
- simple docker container
- on multi container enviroment automatic discovery and syncronisation of tasks
- proxy for file system and s3
- simple http interface
- http path, http header or jwt based tenant discovery 
- configurable jwt role based access control 

Retention is given in minutes from CreationDate or, if a reset retention is called, from RetentionBase.

# Installation

## Docker

simply run 

`docker run -v <host data path>:/data -p 8443:8443 -p 8000:8000 go-blob-store`

to run this service as a single node with a simplefile storage class for storage and a preconfigured fastcache for the cache. Exposes port 8443 for the data interface and 8000 for the metrics.

For other options see the configuration file.

# Configuration

beside the simple default configuration there are some options you might want to change in your enviroment.

## Configuration file

The configuration file service.yaml will be loaded from `/data/config/service.yaml`.

You can simply mount this to another file system and create a new service.yaml with your own configuration. (the defaults as set in the default service.yaml will be used, if the option is not set)

## S3 Storage

The S3 storage provider can be used as main storage or backup storage with the same parameters.

Simply change in engine/storage/ the storage class to S3Storage and add the configured properties:

```yaml
engine:
 retentionManager: SingleRetention
 tenantautoadd: true
 backupsyncmode: false
 storage:
  storageclass: S3Storage
  properties:
   endpoint: "https://192.168.178.45:9002"
   bucket: "goblobstore"
   accessKey: D9Q2D6JQGW1MVCC98LQL
   secretKey: LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr
   password: 4jsfhdjHsd?
   insecure: false
```

you can use the same for the backup storage:

```yaml
engine:
 retentionManager: SingleRetention
 tenantautoadd: true
 backupsyncmode: false
 storage:
  storageclass: SimpleFile
  properties:
   rootpath: /data/storage
 backup:
  storageclass: S3Storage
  properties:
   endpoint: "http://192.168.178.45:9002"
   bucket: "goblobstore"
   accessKey: D9Q2D6JQGW1MVCC98LQL
   secretKey: LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr
   password: 4jsfhdjHsd?
   insecure: false
```

## Fastcache

Fastcache is a specialised storage engine only to be used for a cache storage.

You can combine this with any other storage engine, even with a optional backup engine

```yaml
engine:
 retentionManager: SingleRetention
 tenantautoadd: true
 backupsyncmode: false
 storage:
  storageclass: S3Storage
  properties:
   endpoint: "https://192.168.178.45:9002"
   bucket: "goblobstore"
   accessKey: D9Q2D6JQGW1MVCC98LQL
   secretKey: LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr
   password: 4jsfhdjHsd?
   insecure: false
 cache:
  storageclass: FastCache
  properties:
   rootpath: /data/blobcache
   maxcount: 100000
   maxramusage: 1024000000
```

## Headermapping

There are defined header for operation

tenant: is the name of the tenant in a multi-tenant environment

apikey: the apikey can be used to identify the right usage of this service. (in the configuration you can switch this off)

retention: if set, this blob will die only available until this time (in minutes) is over, counted from the creation time, or, if a reset retention occur, from the reset time.

filename: is the filename of the file itself, and only needed if you use direct binary upload.

The service automatically save all needed headers to the blob description object. If you set a headerprefix, the service will additionally put all header prefixed with this text to the description, too.

In the config you can map the header to other header names. Example:

```yaml
headermapping:
 headerprefix: x-
 retention: X-es-retention
 tenant: X-es-tenant
 filename: X-es-filename
 apikey: X-es-apikey
```

