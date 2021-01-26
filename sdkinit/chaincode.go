package sdkinit

import (
	"fmt"
	mb "github.com/hyperledger/fabric-protos-go/msp"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	contextAPI "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	fabAPI "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	contextImpl "github.com/hyperledger/fabric-sdk-go/pkg/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	lcpackager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/lifecycle"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/test/metadata"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/policydsl"

	"github.com/pkg/errors"
	"log"
	"net/http"
	"strings"
)

var peers = []string{"peer0.core.supply.com", "peer1.core.supply.com", "peer0.f1.supply.com", "peer1.f1.supply.com", "peer0.s1.supply.com", "peer1.s1.supply.com"}

const (
	coreOrg          = "OrgCore"
	s1Org            = "OrgS1"
	f1Org            = "OrgF1"
	chaincodeVersion = "1.0"
	chaincodeName    = "test2"
	label            = chaincodeName + "_" + chaincodeVersion
)

// ① 打包链码
func packageChaincodeV2() []byte {
	desc := &lcpackager.Descriptor{
		Path:  metadata.GetProjectPath() + "/chaincode",
		Type:  pb.ChaincodeSpec_GOLANG,
		Label: label,
	}
	ccPkg, err := lcpackager.NewCCPackage(desc)
	if err != nil {
		panic("can't find code!")
	}
	return ccPkg
}

// ② 安装链码
func installChaincodeV2(info *InitInfo, ccPkg []byte) error {
	installCCReq := resmgmt.LifecycleInstallCCRequest{
		Label:   label,
		Package: ccPkg,
	}
	packageID := lcpackager.ComputePackageID(installCCReq.Label, installCCReq.Package)
	for _, org := range info.Orgs {
		orgPeers, err := DiscoverLocalPeers(*org.OrgAdminClientContext, org.OrgPeerNum)
		if err != nil {
			return fmt.Errorf("DiscoverLocalPeers error: %v", err)
		}
		if flag, _ := checkInstalled(packageID, orgPeers[0], org.OrgResMgmt); flag == false {
			if _, err := org.OrgResMgmt.LifecycleInstallCC(installCCReq, resmgmt.WithTargets(orgPeers...), resmgmt.WithRetry(retry.DefaultResMgmtOpts)); err != nil {
				return fmt.Errorf("LifecycleInstallCC error: %v", err)
			}
			fmt.Println("chaincode success installed on:", org.OrgName)
		}
	}

	return nil
}

// ③ 审批链码
func approveChaincode(packageID string, info *InitInfo) error {
	var mspIDs []string
	for _, org := range info.Orgs {
		mspIDs = append(mspIDs, org.OrgMspId)
	}
	// 审批策略，所有的组织里面的成员
	ccPolicy := policydsl.SignedByNOutOfGivenRole(int32(len(mspIDs)), mb.MSPRole_MEMBER, mspIDs)
	approveCCReq := resmgmt.LifecycleApproveCCRequest{
		Name:              chaincodeName,
		Version:           chaincodeVersion,
		PackageID:         packageID,
		Sequence:          1,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		InitRequired:      true,
	}

	for _, org := range info.Orgs {
		orgPeers, err := DiscoverLocalPeers(*org.OrgAdminClientContext, org.OrgPeerNum)
		fmt.Printf(">>> chaincode approved by %s peers:\n", org.OrgName)
		for _, p := range orgPeers {
			fmt.Printf("	%s\n", p.URL())
		}

		if err != nil {
			return fmt.Errorf("DiscoverLocalPeers error: %v", err)
		}
		if _, err := org.OrgResMgmt.LifecycleApproveCC(info.ChannelID, approveCCReq, resmgmt.WithTargets(orgPeers...), resmgmt.WithOrdererEndpoint(info.OrdererEndpoint), resmgmt.WithRetry(retry.DefaultResMgmtOpts)); err != nil {
			return fmt.Errorf("LifecycleApproveCC error: %v", err)
		}
	}
	return nil
}

// ④ 提交链码
func commitChaincode(info *InitInfo) error {
	var mspIDs []string
	for _, org := range info.Orgs {
		mspIDs = append(mspIDs, org.OrgMspId)
	}
	ccPolicy := policydsl.SignedByNOutOfGivenRole(int32(len(mspIDs)), mb.MSPRole_MEMBER, mspIDs)

	req := resmgmt.LifecycleCommitCCRequest{
		Name:              chaincodeName,
		Version:           chaincodeVersion,
		Sequence:          1,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		InitRequired:      true,
	}
	_, err := info.Orgs[0].OrgResMgmt.LifecycleCommitCC(info.ChannelID, req, resmgmt.WithOrdererEndpoint(info.OrdererEndpoint), resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return fmt.Errorf("LifecycleCommitCC error: %v", err)
	}
	return nil
}

func checkInstalled(packageID string, peer fabAPI.Peer, client *resmgmt.Client) (bool, error) {
	flag := false
	resp1, err := client.LifecycleQueryInstalledCC(resmgmt.WithTargets(peer))
	if err != nil {
		return false, fmt.Errorf("LifecycleQueryInstalledCC error: %v", err)
	}
	for _, t := range resp1 {
		if t.PackageID == packageID {
			flag = true
		}
	}
	return flag, nil
}

func DiscoverLocalPeers(ctxProvider contextAPI.ClientProvider, expectedPeers int) ([]fabAPI.Peer, error) {
	ctx, err := contextImpl.NewLocal(ctxProvider)
	if err != nil {
		return nil, fmt.Errorf("error creating local context: %v", err)
	}

	discoveredPeers, err := retry.NewInvoker(retry.New(retry.TestRetryOpts)).Invoke(
		func() (interface{}, error) {
			peers, serviceErr := ctx.LocalDiscoveryService().GetPeers()
			if serviceErr != nil {
				return nil, fmt.Errorf("getting peers for MSP [%s] error: %v", ctx.Identifier().MSPID, serviceErr)
			}
			if len(peers) < expectedPeers {
				return nil, status.New(status.TestStatus, status.GenericTransient.ToInt32(), fmt.Sprintf("Expecting %d peers but got %d", expectedPeers, len(peers)), nil)
			}
			return peers, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return discoveredPeers.([]fabAPI.Peer), nil
}

func queryInstalledV2() {
}

func installChaincodeV1(sdk *fabsdk.FabricSDK, info *InitInfo) error {
	clientContext := sdk.Context(fabsdk.WithUser(info.OrgAdmin), fabsdk.WithOrg(info.OrgName))
	if clientContext == nil {
		return fmt.Errorf("根据指定的组织名称与管理员创建资源管理客户端Context失败")
	}

	// New returns a resource management client instance.
	rc, err := resmgmt.New(clientContext)
	if err != nil {
		return errors.WithMessage(err, "resmgmt.New error")
	}
	// 按照 GOPATH 找对应文件
	ccPkg, err := gopackager.NewCCPackage(info.CCPath, "/home/tl/go/")
	if err != nil {
		return fmt.Errorf("获取 package 失败")
	}
	// new request of installing chaincode
	req := resmgmt.InstallCCRequest{
		Name:    info.CCName,
		Path:    info.CCPath,
		Version: chaincodeVersion,
		Package: ccPkg,
	}

	for _, peer := range peers {
		if strings.Contains(peer, "f1") {
			fmt.Println("peer change to f1")
			rc, err = newClientWithAdmin(sdk, f1Org)
			if err != nil {
				return errors.WithMessage(err, "resmgmt.New error")
			}
		} else if strings.Contains(peer, "s1") {
			fmt.Println("peer change to s1")
			rc, err = newClientWithAdmin(sdk, s1Org)
			if err != nil {
				return errors.WithMessage(err, "resmgmt.New error")
			}
		} else {
			fmt.Println("peer change to core")
			rc, err = newClientWithAdmin(sdk, coreOrg)
			if err != nil {
				return errors.WithMessage(err, "resmgmt.New error")
			}
		}
		reqPeers := resmgmt.WithTargetEndpoints(peer)
		response, err := rc.InstallCC(req, reqPeers)
		if err != nil {
			log.Printf("Install cc on peer: %s failed, cause: %s", peer, err.Error())
			return errors.WithMessage(err, "installCC error")
		}
		// check other errors
		var errs []error
		for _, resp := range response {
			log.Printf("Install  response status: %v", resp.Status)
			if resp.Info == "already installed" {
				log.Printf("Chaincode %s already installed on peer: %s.\n",
					info.CCName, resp.Target)
				break
			}
			if resp.Status != http.StatusOK {
				errs = append(errs, errors.New(resp.Info))
			}
		}

		if len(errs) > 0 {
			log.Printf("InstallCC errors: %v", errs)
			return errors.WithMessage(errs[0], "installCC first error")
		}
	}
	return nil
}

func instantiateChaincodeV1(sdk *fabsdk.FabricSDK, info *InitInfo) error {
	rc, err := newClientWithAdmin(sdk, "OrgCore")
	if err != nil {
		return errors.WithMessage(err, "resmgmt.New error")
	}
	policy, err := policydsl.FromString("AND ('GylCoreOrg1MSP.member','GylFOrg1MSP.member')")

	instantiateReq := resmgmt.InstantiateCCRequest{
		Name:    info.CCName,
		Path:    info.CCPath,
		Version: chaincodeVersion,
		Policy:  policy,
	}
	reqPeer := resmgmt.WithTargetEndpoints("peer0.core.supply.com")
	resp, err := rc.InstantiateCC(info.ChannelID, instantiateReq, reqPeer)
	if err != nil {
		return errors.WithMessage(err, "instantiate error")
	}
	fmt.Println("peer0.core.supply.com instantiate response:", resp)

	rc, err = newClientWithAdmin(sdk, f1Org)
	if err != nil {
		return errors.WithMessage(err, "instantiate error")
	}
	reqPeer = resmgmt.WithTargetEndpoints("peer0.f1.supply.com")
	resp, err = rc.InstantiateCC(info.ChannelID, instantiateReq, reqPeer)
	if err != nil {
		return errors.WithMessage(err, "instantiate error")
	}
	fmt.Println("peer0.f1.supply.com instantiate response:", resp)
	return nil
}

func newClientWithAdmin(sdk *fabsdk.FabricSDK, org string) (*resmgmt.Client, error) {
	clientContext := sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg(org))
	if clientContext == nil {
		return nil, fmt.Errorf("根据指定的组织名称与管理员创建资源管理客户端Context失败")
	}
	// New returns a resource management client instance.
	rc, err := resmgmt.New(clientContext)
	return rc, err
}

func Chaincode(info *InitInfo) error {
	// V1.x 的链码生命周期
	// 安装链码
	//err := installChaincodeV1(sdk, info)
	//if err != nil {
	//	return err
	//}
	//
	//// 初始化链码
	//err = instantiateChaincodeV1(sdk, info)
	//if err != nil {
	//	return err
	//}
	ccPkg := packageChaincodeV2()
	err := installChaincodeV2(info, ccPkg)
	if err != nil {
		return err
	}
	packageID := lcpackager.ComputePackageID(label, ccPkg)
	err = approveChaincode(packageID, info)
	if err != nil {
		return err
	}
	err = commitChaincode(info)
	if err != nil {
		return err
	}
	return nil
}
