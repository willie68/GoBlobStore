# GoBlobStore
A multi-tenant proxy service for storing binary data in various storage systems with a simple HTTP interface.

features
- multi tenant binary store with strong separation of data
- simple docker container
- on multi container enviroment automatic discovery and syncronisation of tasks
- proxy for file system and s3
- simple http interface
- http path, http header or jwt based tenant discovery 
- configurable jwt role based or basic auth access control 

Retention is given in minutes from CreationDate or, if a reset retention is called, from RetentionBase