# GoBlobStore
A multi-tenant proxy service for storing binary data in various storage systems with a simple HTTP interface.

features
- multi tenant binary store with strong separation of data
- simple docker container
- on multi container environment automatic discovery and synchronization of tasks
- proxy for file system and s3
- simple http interface
- http path, http header or jwt based tenant discovery 
- configurable jwt role based access control
- automatic config substitutio

Retention is given in minutes from CreationDate or, if a reset retention is called, from RetentionBase.

# Installation

## Docker

simply run 

`docker run -v <host data path>:/data -p 8443:8443 -p 8000:8000 go-blob-store`

to run this service as a single node with a simplefile storage class for storage and a preconfigured fastcache for the cache. Exposes port 8443 for the data interface and 8000 for the metrics.

For other options see the configuration file.

# Configuration

beside the simple default configuration there are some options you might want to change in your environment.

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

`tenant`: is the name of the tenant in a multi-tenant environment

`apikey`: the apikey can be used to identify the right usage of this service. (in the configuration you can switch this off)

`retention`: if set, this blob will die only available until this time (in minutes) is over, counted from the creation time, or, if a reset retention occur, from the reset time.

`filename`: is the filename of the file itself, and only needed if you use direct binary upload.

The service automatically save all needed headers to the blob description object. If you set a headerprefix, the service will additionally put all header prefixed with this text to the description, too.

In the config you can map the header to other header names. Example:

```yaml
headermapping:
 headerprefix: x-mcs
 retention: X-mcs-retention
 tenant: X-mcs-tenant
 filename: X-mcs-filename
 apikey: X-mcs-apikey
```

## JWT Tenant discovery and Authorisation

Normally the tenant for the blob storage is discovered by a seperate header. (As you can read in the chapter Headermapping). If you are using JWT for authentication/authorization, the Tenant can be discovered by an extra claim on the jwt. Simply activate jwt authentication with 

```yaml
auth:
 type: jwt
 properties: 
  validate: false
  strict: true
  tenantClaim: Tenant
```

`validate` `false` means, the token is not validated against the issuer. (this is normally ok, when the token is already checked by an api gateway or other serving services) (At the moment this is the only option. JWT Token validation is not implemented.)

`strict` `true` means the call will fail, if not all needed parameters, (at the moment only the tenant) can be evaluated from the token. `false` will fall back to http headers

`tenantClaim` will name the claim name of the tenant value. Defaults to Tenant (optional)

## Index and Search

For finding desired blobs you can configure an index engine. Possible options are

- MongoDB

(sorry, nothing more at this moment)

### Query Language

A separate search language is supported for a search independent of the underlying index engine. This offers a simple syntax for searching.

Some value element examples:

`foo` ~ search the default field for value "foo" in a match or term query

`35` ~ search the default field for the number 35, as an integer in a match or term query

`name:Joe` ~ search the `name` field for the value "Joe" as a match or term query

`count:2` ~ search the `count` field for the numerical value 2 as a match or term query

`msg:"foo bar baz"` ~ search the `msg` field using a match-phrase query

`amount:>=40` ~ search the `amount` field using a range query for documents where the field's value is greater than or equal to 40

`created_at:<2017-10-31T00:00:00Z` ~ search the `created_at` field for dates before Halloween of 2017 (*all datetimes are in RF3339 format, UTC timezone*)

`cash:[50~200]` ~ returns all docs where `cash` field's value is within a range greater than or equal to 50, and less than 200.

`updated_at:[2017-04-22T09:45:00Z~2017-05-03T10:20:00Z]` ~ window ranges can also include RFC3339 UTC datetimes

Any field or parenthesized grouping can be negated with the `NOT` or `!` operator:

`NOT foo` ~ search for documents where default field doesn't contain the token `foo`

`!c:[2017-10-29T00:00:00Z~2017-10-30T00:00:00Z]` ~ returns docs where field `c`'s date value is *not* within the range of October 29-31, 2017 (UTC)

!count:>100` ~ search for documents where `count` field has a value that's *not* greater than 100

`NOT (x OR y)` ~ search the default field for documents that don't contain terms "x" or "y"

Parentheses for grouping of subqueries is not supported:

`NOT foo:bar AND baz:99` ~ return blobs where field `foo`'s value is not "bar" and where field `baz`'s value is 99.

Operators have aliases: `AND` -> `&` and `OR` -> `|`:

### Mongo Index

For the mongo index option you have to provide the following information

```yaml
engine:
...
 index:
  storageclass: MongoDB
  properties:
   hosts: 
	- 127.0.0.1:27017
   username:
   password:
   authdatabase: blobstore
   database: blobstore
```

`username` and `password` can be provided via secret.yaml.

Every tenant will create a collection in the database. For this collection the service will automatically create an index based on the blodID. For direct searching in mongo db simply add an # to the json coded mongo query syntax.
**Attention:** all headers are converted to lower case.

As an example: 

```
#{"$and": [{"x-tenant": "MCS"}, {"x-user": "Willie"} ]}
```

### 

## Tenant

The tenant is the main part to split up the data. Every tenant is based on the tenant name or id. This id should be case insensitive and should only consist of chars which are valid for filenames.
