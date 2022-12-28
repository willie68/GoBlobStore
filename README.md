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
- user defined retention per blob
- index option to search for blob properties

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

Every entry of the configuration can be set with a ${} macro for using environment variables for the configuration.

## Storagesystem

The configuration of the storage system consists of several providers. First the primary provider called `storage`

The data is stored here first and forms the basis and reference for the entire storage.

Second, the `backup`: each blob is stored in the backup after successful storage in the primary storage (synchronous or asynchronous). If a failure occurs in the primary storage, then the blob can be restored from the backup, automatically or manually.

a `cache` can be configured as a third component. This is also operated automatically. An adjustable (number or size) amount of blobs can be kept in the cache for quick access.

The storage providers in turn can be used for all 3 parts. Exceptions are, of course, specialized providers such as FastCache.

Another unit is the retention system. This works independently of the Storgae providers.

The `index` is the fifth element. This is used to search for specific blobs via properties, be they system or custom properties.

### Tenant specific backup storage

for every tenant there is the possibility to allow a specific backup storage system, beside of using the default backup storage. This feature will be activate with the `allowtntbackup` flag in the configuration. Now a tenant have the possibility to add a own backup storage to the configuration. After creating (or changing) this storage, the blobstore will automatically start a resynchronization task for that tenant. This task will automatically move all backup file to the new storage provider. In the time of this operation further changes on the backup store will be permitted. Allowed provider for the tenant backup storage are: S3 Storage, Blob Storage. 

#### Limitation:

- Only S3 Storage is permitted

- No encryption for external S3 storage

- no auto move on changes of the storage properties

   

Example of a post to create a new tenant backup storage:
POST: https://localhost:8443/api/v1/config
Headers: X-tenant: <tenant>

```json
{
  "storageclass": "S3Storage",
  "properties" : {
   "endpoint": "http://127.0.0.1:9001",
   "bucket": "mcs",
   "accessKey": "xiSwpTnOf6QXxu3Y",
   "secretKey": "sT7lJIgV4tYoOljdpfr9kMoLE0PgMPJ9"
  }
}
```

Answer:

```json
{
  "type": "createResponse",
  "tenantid": "mcs",
  "backup": "S3Storage"
}
```

## SimpleFile Storage

The simple file storage is a file system based storage. 

```yaml
engine:
 retentionManager: SingleRetention
 tenantautoadd: true
 backupsyncmode: false
 allowtntbackup: false
 storage:
  storageclass: SimpleFile
  properties:
   rootpath: /data/storage
```

You can use this storage for all kind of storage types, (even backup or cache). The only property needed is the rootpath which will lead to the used file system. On docker you can use any mount point / volume for that. Every tenant will get a subfolder. On this tenant directory there will be a 2 dimensional folder structure for  the blob data. For the retention files there will be a dedicated folder.

## S3 Storage

The S3 storage provider can be used as main storage or backup storage with the same parameters.

Simply change in engine/storage/ the storage class to S3Storage and add the configured properties:

```yaml
engine:
 retentionManager: SingleRetention
 tenantautoadd: true
 backupsyncmode: false
 allowtntbackup: false
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
 allowtntbackup: false
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
 allowtntbackup: false
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

`retention`: (optional) if set, this blob will die only available until this time (in minutes) is over, counted from the creation time, or, if a reset retention occur, from the reset time.

`filename`: is the filename of the file itself, and only needed if you use direct binary upload.

`blobid`: (optional) is the predefined blob id of the file. This must be tenant unique otherwise you will get an conflict error.

The service automatically save all needed headers to the blob description object. If you set a headerprefix, the service will additionally put all header prefixed with this text to the description, too.

In the config you can map the header to other header names. Example:

```yaml
headermapping:
 headerprefix: x-mcs
 retention: X-mcs-retention
 tenant: X-mcs-tenant
 filename: X-mcs-filename
 apikey: X-mcs-apikey
 blobid: X-mcs-blobid
```

## JWT Tenant discovery and Authorization

Normally the tenant for the blob storage is discovered by a seperate header. (As you can read in the chapter Header mapping). If you are using JWT for authentication/authorization, the Tenant can be discovered by an extra claim on the jwt. Simply activate jwt authentication with 

```yaml
auth:
 type: jwt
 properties: 
  validate: false
  strict: true
  tenantClaim: Tenant
  roleClaim: 
  rolemapping: 
   object-reader: 
   object-creator:
   object-admin:
   tenant-admin:
   admin:
```

`validate` `false` means, the token is not validated against the issuer. (this is normally ok, when the token is already checked by an api gateway or other serving services) (At the moment this is the only option. JWT Token validation is not implemented.)

`strict` `true` means the call will fail, if not all needed parameters, (at the moment only the tenant) can be evaluated from the token. `false` will fall back to http headers

`tenantClaim` will name the claim name of the tenant value. Defaults to Tenant (optional)

### Authorization Roles

In the blob storage system there are some small roles for the different parts of the blob storage service. Roles can only be used with JWT and role mapping activated. You can deactivate the role checking, if you left the roleClaim property empty.

| Role name      | What the user with this role can do.                         |
| -------------- | ------------------------------------------------------------ |
| object-reader  | A user with this role can only read the data from his tenant. <br />And can do a search and list objects. |
| object-creator | A user with this role can create new blobs. And only this. <br />No view or list permissions are granted |
| object-admin   | A user with this role can view, create and delete objects. <br />And he can set/modify object properties, like metadata and retention. |
| tenant-admin   | A user with this role can manage the tenant properties<br />(at the moment not implemented), <br />do check and restore for the whole storage |
| admin          | A user with this role can manage the service itself, as <br />adding/deleting new tenants to the service. <br />With this role only, you can't write, read objects from any tenant. |

Example with full role mapping:

```yaml
auth:
 type: jwt
 properties: 
  validate: true
  strict: true
  tenantClaim: Tenant
  roleClaim: Roles
  rolemapping: 
   object-reader: Reader
   object-creator: Writer
   object-admin: ObAdmin
   tenant-admin: TnAdmin
   admin: Admin
```



## Index and Search

For finding desired blobs you can configure an index engine. Possible options are

- MongoDB
- Bluge (https://blugelabs.com/bluge/) (only in single node installations)

(sorry, nothing more at this moment)

### Query Language

A separate query language is supported for search independency of the underlying index engine. This offers a simple syntax for searching.

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

### Internal Fulltextindex

For smaller installations there is a small fulltext implementation based on bluge. (https://blugelabs.com/) This index can only be used in a single instance installation. With multi instances the writing can fail, if the index of a tenant is written from two nodes at a time. For multi instance searching please use the mongo index.

```yaml
engine:
...
 index:
  storageclass: bluge
  properties:
    rootpath: <path to a folder>
```

You can set the root path to the same folder as in the SimpleFile storage. The structure will be integrated. For every tenant there will be a subfolder. And in this tenant directory for this index there will be a folder _idx created, which will store all needed files for the index. 

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

## Tenant Based API Endpoints

The tenant is the main part to split up the data. Every tenant is based on the tenant name or id. This id should be case insensitive and should only consist of chars which are valid for filenames.

The tenant can be given in different ways.

First via subpath:
the main routes are:
`/api/v1/stores/{tntid}/blobs/` 

`/api/v1/stores/{tntid}/search/`

where tntid id is the id of the tenant. 

Second via jwt: you can have a configurable claim for the tenant, which is used in every tenant based call. So than you can use the following routes. The config and admin routes are intentionally only avaible without tenant subfolder.

`/api/v1/blobs/`
`/api/v1/search/`
`/api/v1/config/`
`/api/v1/config/stores/`
`/api/v1/admin/check`
`/api/v1/admin/restore`

The third option is a configurable http header. This order is also the order for evaluating. With one exclusion, if you try to select the tenant via route and jwt tenant evaluation is active, than both tenants will be checked to be equal. Otherwise access is denied.
