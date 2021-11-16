The blobstorage consist of some storage implementations and a business manager.

## dao/simplefile

This is the implementation of a simple file storage. 

Every tenant has it's own folder. 

Retention files are stored in a seperate folder called `retention`.

The binary file is stored into a double hierachical folder structure, as the first 2 digits of the id are the first folder, third and forth digits represent the second subfolder. File name is the id and the postfix `.bin`

Because of some compatibility with older java services, reading can be done in a simple folder structure. So the base folder is the tenant.

The description is placed beside the binary and has the extension `.json`. 

## dao/s3

The implementation of the s3 storage provider.

tenants are placed into a separate file called `storelist.json`. Every file is automatically encrypted.

every tenant got his own base folder.

Retention files are stored in a seperate folder called `retention`.

The binary file is stored into a double hierarchical folder structure, as the first 2 digits of the id are the first folder, third and forth digits represent the second sub folder. File name is the id and the postfix `.bin`

The description is placed as user properties the the file. 



## dao/business

Here is the implementation of the business part of the storage. The mainstorage class handles the usages of backup and cached storage as the base storage class.



## dao/retentionmanager

Because of some circle dependencies the retention manager class must be in the main dao folder. At the moment only the single node retention manager is implemented, which will take control over all retention related parts. It can consist with other single retention manager nodes, but they will not share any workload. Every retention manager will have a full list of all retentions of the complete system. So on a multi node setup,  there can be some errors present because f missing retention files (because another retention manager was faster on deletion)