package main

import (
	"log"
	"github.com/serg666/accountstorage/chaincode"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(new(chaincode.AccountStorage))
	if err != nil {
		log.Panicf("Error creating accountstorage chaincode: %v", err)
	}

	if err := cc.Start(); err != nil {
		log.Panicf("Error starting accountstorage chaincode: %v", err)
	}
}
