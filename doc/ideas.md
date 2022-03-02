# Ideas for the blob storage

Here you will find my ideas to the next implementation parts of the blob storage.



## Roles

In the blob storage system there should be some small roles for the different parts of the blob storage service. Roles can only be used with JWT activated.

| Role name      | What the user with this role can do.                         |
| -------------- | ------------------------------------------------------------ |
| object-reader  | A user with this role can only read the data from his tenant. <br />And can do a search and list objects. |
| object-creator | A user with this role can create new blobs. And only this. <br />No view or list permissions are granted |
| object-admin   | A user with this role can view, create and delete objects. <br />And he can set/modify object properties, like metadata and retention. |
| tenant-admin   | A user with this role can manage the tenant properties<br />(at the moment not implemented), <br />do check and restore for the whole storage |
| admin          | A user with this role can manage the service itself, as <br />adding/deleting new tenants to the service. <br />With this role only, you can't write, read objects from any tenant. |

