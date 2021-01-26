package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
)

func checkInit(t *testing.T, stub *shimtest.MockStub, args [][]byte) {
	result := stub.MockInit("1", args)

	if result.Status != shim.OK {
		fmt.Println("Init fail, errMsg: ", string(result.Message))
		t.FailNow()
	}
}

func initIssue(t *testing.T, stub *shimtest.MockStub, issuer string, id string, amount string, endDate string, contractHash string, invoiceHash string) {
	result := stub.MockInvoke("1", [][]byte{[]byte("issue"), []byte(issuer), []byte(id), []byte(amount), []byte(endDate), []byte(contractHash), []byte(invoiceHash)})

	if result.Status != shim.OK {
		fmt.Println("Invoke Issue error, errMsg: ", result.Message)
		t.FailNow()
	}
}

func checkIssue(t *testing.T, stub *shimtest.MockStub, args [][]byte) {
	result := stub.MockInvoke("1", args)

	if result.Status != shim.OK {
		fmt.Println("Invoke fail, errMsg: ", result.Message)
		t.FailNow()
	}
}

func checkQueryByID(t *testing.T, stub *shimtest.MockStub, id string, voucherAsset *VoucherAsset) {
	var result = stub.MockInvoke("1", [][]byte{[]byte("queryByID"), []byte(id)})

	if result.Status != shim.OK {
		fmt.Println("Query VoucherAsset fail, ID=", id)
		t.FailNow()
	}

	if voucherAsset == nil && result.Payload != nil {
		fmt.Println("Query VoucherAsset fail, ID=", id, " expected is", voucherAsset)
		t.FailNow()
	}

	if voucherAsset != nil && result.Payload == nil {
		fmt.Println("Query VoucherAsset fail, cannot find ID=", id, " voucherAsset!")
		t.FailNow()
	}

	resultVoucherAsset := new(VoucherAsset)
	json.Unmarshal(result.Payload, resultVoucherAsset)
	if voucherAsset.Issuer != resultVoucherAsset.Issuer || voucherAsset.Amount != resultVoucherAsset.Amount {
		fmt.Println("Query VoucherAsset is=", resultVoucherAsset, ", but expected is=", voucherAsset)
		t.FailNow()
	}
}

func Test_Init(t *testing.T) {
	chaincode := new(VoucherAssetChaincode)
	stub := shimtest.NewMockStub("Test_Init", chaincode)

	checkInit(t, stub, [][]byte{[]byte("a")})
}

func Test_issue(t *testing.T) {
	chaincode := new(VoucherAssetChaincode)
	stub := shimtest.NewMockStub("Test_issue", chaincode)

	initIssue(t, stub, "a", "123456", "10000", "2021-12-30", "hash-123", "hash-456")

	voucherAsset := VoucherAsset{
		Issuer: "a",
		Amount: 10000,
	}

	checkQueryByID(t, stub, "123456", &voucherAsset)
}
