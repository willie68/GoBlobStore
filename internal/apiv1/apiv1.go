package apiv1

import "fmt"

// APIVersion the actual implemented api version
const APIVersion = "1"

// BaseURL is the url all endpoints will be available under
var BaseURL = fmt.Sprintf("/api/v%s", APIVersion)

// APIKey the apikey of this service
var APIKey string

const adminSubpath = "/admin"
const storesSubpath = "/stores"
const configSubpath = "/config"
const blobsSubpath = "/blobs"
const searchSubpath = "/search"
