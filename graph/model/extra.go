package model

//
// Errors
//

type GqlErrors struct {
    Errors []GqlError `json:"errors"`
}

type GqlError struct {
    Location string  `json:"location"`
    Message string  `json:"message"`
}

// JsonAtom is a general interface
// for decoding unknonw structure
type JsonAtom = map[string]interface{}

//
// Utils
//

var TensionHookPayload string = `{
  uid
  Post.createdBy { User.username }
  Tension.action
  Tension.emitter {
    Node.nameid
  }
  Tension.receiver {
    Node.nameid
    Node.isPrivate
    Node.charac {
        NodeCharac.userCanJoin
        NodeCharac.mode
    }
  }
  Tension.blobs %s {
    uid
    Blob.blob_type
    Blob.md
    Blob.node {
      NodeFragment.name
      NodeFragment.nameid
      NodeFragment.type_
      NodeFragment.about
      NodeFragment.mandate {
        expand(_all_)
      }

      NodeFragment.first_link
      NodeFragment.second_link
      NodeFragment.skills
      NodeFragment.role_type

      NodeFragment.children {
        NodeFragment.first_link
        NodeFragment.role_type
      }
    }

  }
}`

