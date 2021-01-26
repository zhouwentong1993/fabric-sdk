package sdkinit

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

const ChaincodeVersion = "1.0"

func SetupSDK(ConfigFile string, initialized bool, info *InitInfo) (*fabsdk.FabricSDK, error) {

	if initialized {
		return nil, fmt.Errorf("Fabric SDK已被实例化")
	}

	sdk, err := fabsdk.New(config.FromFile(ConfigFile))
	if err != nil {
		return nil, fmt.Errorf("实例化Fabric SDK失败: %v", err)
	}

	for _, org := range info.Orgs {
		org.orgMspClient, err = mspclient.New(sdk.Context(), mspclient.WithOrg(org.OrgName))
		if err != nil {
			return nil, err
		}
		orgContext := sdk.Context(fabsdk.WithUser(org.OrgAdminUser), fabsdk.WithOrg(org.OrgName))
		org.OrgAdminClientContext = &orgContext

		// New returns a resource management client instance.
		resMgmtClient, err := resmgmt.New(orgContext)
		if err != nil {
			return nil, fmt.Errorf("根据指定的资源管理客户端Context创建通道管理客户端失败: %v", err)
		}
		org.OrgResMgmt = resMgmtClient
	}

	fmt.Println("Fabric SDK初始化成功")
	return sdk, nil
}

func QueryLedger(sdk *fabsdk.FabricSDK, info *InitInfo) error {
	org1AdminChannelContext := sdk.ChannelContext(info.ChannelID, fabsdk.WithUser(info.OrgAdmin), fabsdk.WithOrg(info.OrgName))

	// Ledger client
	client, err := ledger.New(org1AdminChannelContext)
	if err != nil {
		return fmt.Errorf("Failed to create new resource management client: %s", err)
	}

	ledgerInfo, err := client.QueryInfo()
	if err != nil {
		return fmt.Errorf("QueryInfo return error: %s", err)
	}
	fmt.Println("查询账本数据为：", ledgerInfo)
	block, err1 := client.QueryBlockByHash(ledgerInfo.BCI.CurrentBlockHash)
	if err1 != nil {
		return fmt.Errorf("QueryBlock return error: %s", err1)
	}
	fmt.Println("查询的区块数据为：", block)
	return nil
}