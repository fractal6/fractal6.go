package email

// Export unexported symbols for testing.
type DetailsExtension = detailsExtension

var DetailsBodyStyle = detailsBodyStyle

// PostalAttachment is the JSON-tagged Postal payload entry, exposed for tests
// that need to assert the wire shape (e.g. content_id omission for Bucket B).
type PostalAttachment = postalAttachment
