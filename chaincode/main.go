package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	medicalContract := new(MedicalContract)
	cc, err := contractapi.NewChaincode(medicalContract)
	if err != nil {
		log.Panicf("创建医疗链码实例失败: %v", err)
	}

	if err := cc.Start(); err != nil {
		log.Panicf("启动医疗链码服务失败: %v", err)
	}
}
