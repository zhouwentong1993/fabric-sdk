package sdkinit

import (
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	contextAPI "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
)

type OrgInfo struct {
	OrgAdminUser          string
	OrgName               string
	OrgMspId              string
	OrgUser               string
	orgMspClient          *mspclient.Client
	OrgAdminClientContext *contextAPI.ClientProvider
	OrgResMgmt            *resmgmt.Client
	OrgPeerNum            int
}

type InitInfo struct {
	Orgs            []*OrgInfo
	ChannelID       string
	ChannelConfig   string
	OrgAdmin        string
	OrgName         string
	OrdererEndpoint string
	OrgResMgmt      *resmgmt.Client
	CCName          string
	CCPath          string
}
