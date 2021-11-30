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