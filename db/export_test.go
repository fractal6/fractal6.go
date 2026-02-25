package db

// Export unexported symbols for testing.
var DqlQueries = dqlQueries
var DqlMutations = dqlMutations

// DecodeDqlResp exports decodeDqlResp for testing.
var DecodeDqlResp = decodeDqlResp

// Export decode helpers for unit testing.
var CleanDqlKey = cleanDqlKey
var DecodeField = decodeField
var DecodeSubField = decodeSubField
var DecodeSubSubField = decodeSubSubField
