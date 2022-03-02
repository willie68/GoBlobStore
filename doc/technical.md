The blobstorage consist of some storage implementations and a business manager.

## dao/simplefile

This is the implementation of a simple file storage. 

Every tenant has it's own folder. 

Retention files are stored in a separate folder called `retention`.

The binary file is stored into a double hierarchical folder structure, as the first 2 digits of the id are the first folder, third and forth digits represent the second subfolder. File name is the id and the postfix `.bin`

Because of some compatibility with older java services, reading can be done in a simple folder structure. So the base folder is the tenant.

The description is placed beside the binary and has the extension `.json`. 

### Parameters

`rootpath`: path to the file system to store the data

## dao/s3

The implementation of the s3 storage provider.

tenants are placed into a separate file called `storelist.json`. Every file is automatically encrypted.

every tenant got his own base folder.

Retention files are stored in a separate folder called `retention`.

The binary file is stored into a double hierarchical folder structure, as the first 2 digits of the id are the first folder, third and forth digits represent the second sub folder. File name is the id and the postfix `.bin`

The description is placed as user properties the the file. 

### Parameters

`endpoint`: URL to the s3 endpoint

`insecure`: bool true for not checking the SSL certificate (mainly for self signed certificates)

`bucket`: the bucket to use

`accessKey`: the access key of the s3 storage

`secretKey`: the secret key of the s3 storage

`password`: a salt to the password encryption part for storing encryption

## dao/fastcache

The implementation of cache storage provider. This implementation usage a LRU cache mechanism. The description files are stored in memory for all cached blobs. The files are stored on a separate file system (fast bound SSD Storage or similar). If the file size <mffrs the file is stored into the memory, too. In the option you can define, how many files are stored into the cache and file system. And you can define the max ram usage for the in memory stored files. Both will be checked automatically. 

### Parameters

`rootpath`: path to the filesystem to store the cached data

`maxcount`: how many files are stored into the cache

`maxramusage`: for files <mffrs max RAM usage.

`maxfilesizeforram`: file size to put into the in memory cache (mffrs)

## dao/business

Here is the implementation of the business part of the storage. The mainstorage class handles the usages of backup and cached storage as the base storage class.



## dao/retentionmanager

Because of some circle dependencies the retention manager class must be in the main dao folder. At the moment only the single node retention manager is implemented, which will take control over all retention related parts. It can consist with other single retention manager nodes, but they will not share any workload. Every retention manager will have a full list of all retentions of the complete system. So on a multi node setup,  there can be some errors present because f missing retention files (because another retention manager was faster on deletion)

# FastCache

The FastCache is an LRU implementation with 2-level data storage. All files in the cache are stored on a separate volume. This should be a very fast local medium. (e.g. local SSD) Files up to a certain file size (100kb) are also stored in the RAM.

First you set the maximum number of files in the cache with an option. These are all stored on the assigned volume.

In addition, a memory size is specified that specifies how much memory the files stored in the memory can use. Thus, small files can be served directly from the memory, which brings additional performance.

Why LRU? There is a corresponding note here: https://dropbox.tech/infrastructure/caching-in-theory-and-practice

A bloom filter is also used to determine whether a file is in the cache. Thus, the CacheMiss case can be decided quickly in most cases (the setting is 0.1%).

# Search/Index

The first implemented index engine is mongodb.

For parsing the query string i use https://github.com/mna/pigeon and a PEG grammar based on https://github.com/elireisman/go_es_query_parser

This is one of the moving targets. You will find the peg file in build/pigeon/parser.peg

There is much more implemented in the parser file, but not everything is working. So i only documented the working parts in the readme. In addition, various automated tests are still missing. Here, above all, the evaluation of the results. The parser will be generated into pkg/model/query

Be ware the parser itself is not thread safe, so a serialization is done in the API.