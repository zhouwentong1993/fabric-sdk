package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// VoucherAssetChaincode : 票据资产 链码
type VoucherAssetChaincode struct {
}

// Status enum for commercial paper state property
type Status uint

const (
	// ISSUED status for when a VoucherAsset has been issued
	ISSUED Status = iota + 1
	// TRADING status for when a VoucherAsset is trading
	TRADING
	// SPLIT status for when a VoucherAsset is spliting
	SPLIT
	// REDEEMED status for when a VoucherAsset has been redeemed
	REDEEMED
)

// VoucherAsset : 票据资产
type VoucherAsset struct {
	ID           string    `json:"id"`
	ParentID     string    `json:"parentId"`
	Issuer       string    `json:"issuer"`
	Owner        string    `json:"owner"`
	Amount       int       `json:"amount"`
	CreateDate   time.Time `json:"createDate"`
	EndDate      time.Time `json:"endDate"`
	ContractHash string    `json:"contractHash"`
	InvoiceHash  string    `json:"invoiceHash"`
	Status       Status    `json:"status"`
	// Signature    string    `json:"signature"`
}

// QueryResult 查询结果集
type QueryResult struct {
	Key    string        `json:"key"`
	Record *VoucherAsset `json:"record"`
}

func (status Status) String() string {
	names := []string{"ISSUED", "TRADING", "SPLIT", "REDEEMED"}

	if status < ISSUED || status > REDEEMED {
		return "UNKNOWN"
	}

	return names[status-1]
}

// Init 初始化方法
func (t *VoucherAssetChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("VoucherAsset Init.")

	_, args := stub.GetFunctionAndParameters()

	fmt.Printf("args: %v\n", args)

	return shim.Success(nil)
}

// Invoke 调用方法
func (t *VoucherAssetChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	function, args := stub.GetFunctionAndParameters()

	if function == "issue" {
		return t.issue(stub, args)
	} else if function == "transfer" {
		return t.transfer(stub, args)
	} else if function == "queryByID" {
		return t.queryByID(stub, args)
	} else if function == "queryAll" {
		return t.queryAll(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"issue\" \"transfer\" \"queryByID\" \"queryAll\"")
}

// issue 发行票据
// "Args": [\"a\",\"123456\",\"10000\",\"2021-12-31\",\"123456\",\"654321\"]
func (t *VoucherAssetChaincode) issue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 6 {
		errMsg := "Issue function args length must be 6!"
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	// TODO: warning: 获取当前时间这里是一个坑
	txTimestamp, _ := stub.GetTxTimestamp()

	issuer := args[0]
	owner := args[0]
	id := args[1]
	amount, _ := strconv.Atoi(args[2])
	createDate, _ := ptypes.Timestamp(txTimestamp)
	endDate, _ := time.Parse("2006-01-02", args[3])
	contractHash := args[4]
	invoiceHash := args[5]

	// 判断 发行的票据资产金额必须大于0
	if amount <= 0 {
		errMsg := fmt.Sprintf("issue: the amount=%d must greater then 0!", amount)
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	// 判断 到期日期必须要大于创建日期
	if createDate.After(endDate) {
		errMsg := fmt.Sprintf("issue: the createDate=%s must before endDate=%s", createDate.Format("2006-01-01"), endDate.Format("2006-01-01"))
		fmt.Printf(errMsg)
		return shim.Error(errMsg)
	}

	voucherAsset := VoucherAsset{
		Issuer:       issuer,
		Owner:        owner,
		ID:           id,
		Amount:       amount,
		CreateDate:   createDate,
		EndDate:      endDate,
		ContractHash: contractHash,
		InvoiceHash:  invoiceHash,
		Status:       ISSUED,
	}

	// 保存到本地账本
	voucherAssetAsBytes, _ := json.Marshal(voucherAsset)

	fmt.Printf("issue: new asset=%s\n", string(voucherAssetAsBytes))

	err := stub.PutState(id, voucherAssetAsBytes)

	if err != nil {
		errMsg := err.Error()
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	return shim.Success(voucherAssetAsBytes)
}

// transfer 转账
// a 向 b 转账5000分，原始ID为 id-123456
// "Args": [\"id-123456\",\"a\",\"b\",\"5000\",\"contractHash-123456\",\"invoiceHash-123456\"]
func (t *VoucherAssetChaincode) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 6 {
		var errMsg = "transfer: args length must be 6!"
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	originID := args[0]
	originOwner := args[1]
	newOwner := args[2]
	// 转账金额，单位：分
	amount, _ := strconv.Atoi(args[3])
	contractHash := args[4]
	invoiceHash := args[5]

	// 判断 原owner和新owner不相同
	if originOwner == newOwner {
		var errMsg = fmt.Sprintf("transfer: the originOwner=%s is equal to newOwner=%s", originOwner, newOwner)
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	// 判断 转账金额amount必须要大于0
	if amount <= 0 {
		errMsg := "transfer: the amount must greater then 0"
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	// 获取 原始票据资产
	voucherAsBytes, err := stub.GetState(originID)

	if err != nil {
		fmt.Printf("transfer: Failed to read from world state, key=%s, errMsg=%s\n", originID, err.Error())
		return shim.Error(err.Error())
	}

	if voucherAsBytes == nil {
		errMsg := fmt.Sprintf("transfer: the originID=%s is illegal.", originID)
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	originVoucherAsset := new(VoucherAsset)
	_ = json.Unmarshal(voucherAsBytes, originVoucherAsset)

	// 判断 原始账户的owner为给定的originOwner
	if originVoucherAsset.Owner != originOwner {
		var errMsg string
		errMsg = fmt.Sprintf("transfer: the originOwner=%s not equal arg owner=%s", originVoucherAsset.Owner, originOwner)
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	// 判断 转账金额不大于原有金额
	if amount > originVoucherAsset.Amount {
		var errMsg = fmt.Sprintf("transfer: The transfer amount=%d is greater then origin amount=%d", amount, originVoucherAsset.Amount)
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	// 先更新原始票据资产状态为 SPLIT
	originVoucherAsset.Status = SPLIT
	originVoucherAssetAsBytes, _ := json.Marshal(originVoucherAsset)

	if err := stub.PutState(originVoucherAsset.ID, originVoucherAssetAsBytes); err != nil {
		fmt.Println(err.Error())
		return shim.Error(err.Error())
	}

	// 生成新的票据资产
	assetList := list.New()

	assetList.PushBack(
		VoucherAsset{
			Issuer:       originVoucherAsset.Issuer,
			Owner:        newOwner,
			ParentID:     originID,
			ID:           originID + "_" + newOwner,
			Amount:       amount,
			CreateDate:   originVoucherAsset.CreateDate,
			EndDate:      originVoucherAsset.EndDate,
			ContractHash: contractHash,
			InvoiceHash:  invoiceHash,
			Status:       TRADING,
		},
	)

	if originVoucherAsset.Amount > amount {
		assetList.PushBack(
			VoucherAsset{
				Issuer:       originVoucherAsset.Issuer,
				Owner:        originOwner,
				ParentID:     originID,
				ID:           originID + "_" + originOwner,
				Amount:       originVoucherAsset.Amount - amount,
				CreateDate:   originVoucherAsset.CreateDate,
				EndDate:      originVoucherAsset.EndDate,
				ContractHash: contractHash,
				InvoiceHash:  invoiceHash,
				Status:       TRADING,
			},
		)
	}

	// 保存到本地账本
	for e := assetList.Front(); e != nil; e = e.Next() {
		voucherAssetAsBytes, _ := json.Marshal(e.Value)
		fmt.Println(string(voucherAssetAsBytes))
		val, ok := e.Value.(VoucherAsset)
		if !ok {
			var errMsg = "transfer: zhe type not match."
			fmt.Println(errMsg)
			return shim.Error(errMsg)
		}
		err := stub.PutState(val.ID, voucherAssetAsBytes)
		if err != nil {
			fmt.Println(err.Error())
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}

// queryByID 根据票据ID查询票据信息
func (t *VoucherAssetChaincode) queryByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) != 1 {
		errMsg := fmt.Sprintf("queryByID: args length must 1.")
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	voucherID := args[0]

	voucherAsBytes, err := stub.GetState(voucherID)

	if err != nil {
		errMsg := fmt.Sprintf("queryByID: Failed to read from world state. '%s'", err.Error())
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	if voucherAsBytes == nil {
		var errMsg = fmt.Sprintf("queryByID: '%s' does not exist", voucherID)
		fmt.Println(errMsg)
		return shim.Error(errMsg)
	}

	return shim.Success(voucherAsBytes)
}

// queryAll 查询所有账本信息
func (t *VoucherAssetChaincode) queryAll(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	startKey := ""
	endKey := ""

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)

	if err != nil {
		fmt.Println(err.Error())
		return shim.Error(err.Error())
	}

	defer resultsIterator.Close()

	results := []QueryResult{}

	for resultsIterator.HasNext() {
		queryResp, err := resultsIterator.Next()

		if err != nil {
			fmt.Println(err.Error())
			return shim.Error(err.Error())
		}

		voucherAsset := new(VoucherAsset)
		_ = json.Unmarshal(queryResp.Value, voucherAsset)

		queryResult := QueryResult{Key: queryResp.Key, Record: voucherAsset}

		results = append(results, queryResult)
	}

	resultsAsBytes, _ := json.Marshal(results)

	return shim.Success(resultsAsBytes)
}

func main() {
	err := shim.Start(new(VoucherAssetChaincode))

	if err != nil {
		fmt.Printf("Error starting VoucherAsset chaincode: %s\n", err)
	}
}
