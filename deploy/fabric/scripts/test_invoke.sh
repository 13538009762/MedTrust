#!/bin/bash
ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt
PEER0_ORG1_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
PEER0_ORG2_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

echo ">>> [1] 提交交易 CreateMedicalRecord..."
peer chaincode invoke -o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA -C medchannel -n medical --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles $PEER0_ORG1_CA --peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles $PEER0_ORG2_CA -c '{"Args":["MedicalContract:CreateMedicalRecord","REC_TEST_001","QmTest12345","hashabc123","1","EMR","2026-09-13T12:00:00Z"]}'

sleep 2

echo ">>> [2] 查询交易 QueryMedicalRecord..."
peer chaincode query -C medchannel -n medical -c '{"Args":["MedicalContract:QueryMedicalRecord","REC_TEST_001"]}'
