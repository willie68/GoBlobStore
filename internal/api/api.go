package api

// TenantHeaderKey in this header the right tenant should be inserted
const TenantHeaderKey = "tenant"

// APIKeyHeaderKey in this header the right api key should be inserted
const APIKeyHeaderKey = "apikey"

// RetentionHeaderKey is the header for defining a retention time
const RetentionHeaderKey = "retention"

// FilenameKey key for the headermapping for the file name
const FilenameKey = "filename"

//BlobIDHeaderKey  is the header for defining a blob id
const BlobIDHeaderKey = "blobid"

// HeaderPrefixKey all headers with this prefix will be saved, too
const HeaderPrefixKey = "headerprefix"

// URLParamTenantID url parameter for the tenant id
const URLParamTenantID = "tntid"

// MetricsEndpoint endpoint subpath  for metrics
const MetricsEndpoint = "/metrics"
