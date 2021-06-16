package contact

import (
	. "github.com/unity-go/util"
)

func NewContact(node NodeID, address string, id int) &api.Contact {
	return &api.Contact{
		ID:       node,
		Address:  address,
		QuorumID: id,
	}
}
