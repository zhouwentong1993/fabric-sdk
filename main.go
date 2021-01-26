package main

import (
	"fabric-sdk/sdkinit"
	"fmt"
)

const (
	local       = true
	initialized = false
)

var configFile = "/home/tl/fabric-sdk/config.yaml"
var prefix = ""

func main() {
	if local {
		configFile = "/Users/finup123/GolandProjects/fabric-sdk/config-local.yaml"
		prefix = "/Users/finup123/GolandProjects/fabric-sdk/"
	} else {
		configFile = "/home/tl/fabric-sdk/config.yaml"
		prefix = "/home/tl/fabric-sdk/"
	}
	orgs := []*sdkinit.OrgInfo{
		{
			OrgAdminUser: "Admin",
			OrgName:      "OrgCore",
			OrgMspId:     "GylCoreOrg1MSP",
			OrgUser:      "User1",
			OrgPeerNum:   2,
		},
		{
			OrgAdminUser: "Admin",
			OrgName:      "OrgF1",
			OrgMspId:     "GylFOrg1MSP",
			OrgUser:      "User1",
			OrgPeerNum:   2,
		},
		{
			OrgAdminUser: "Admin",
			OrgName:      "OrgS1",
			OrgMspId:     "GylSOrg1MSP",
			OrgUser:      "User1",
			OrgPeerNum:   2,
		},
	}
	initInfo := &sdkinit.InitInfo{

		ChannelID:     "test2channel",
		ChannelConfig: prefix + "test2channel.tx",

		OrgAdmin:        "Admin",
		OrgName:         "OrgCore",
		OrdererEndpoint: "orderer0.supply.com",

		CCName: "supply",
		CCPath: "github.com/hyperledger/fabric/deploy/chaincodes/chaincode",

		Orgs: orgs,
	}

	sdk, err := sdkinit.SetupSDK(configFile, initialized, initInfo)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}

	defer sdk.Close()

	// 创建并加入通道
	//err = sdkinit.CreateChannel(sdk, initInfo)
	//if err != nil {
	//	fmt.Printf(err.Error())
	//	return
	//}

	// 安装链码
	err = sdkinit.Chaincode(initInfo)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}
	// 查询账本
	err = sdkinit.QueryLedger(sdk, initInfo)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}

}
