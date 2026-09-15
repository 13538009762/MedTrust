#!/bin/bash
set -e

CHANNEL_NAME="medchannel"
CC_NAME="medical"
CC_VERSION="1.0"
CC_SEQUENCE=1

ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt
PEER0_ORG1_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
PEER0_ORG2_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

setGlobalsOrg1() {
  export CORE_PEER_LOCALMSPID="Org1MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
  export CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
  export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
}

setGlobalsOrg2() {
  export CORE_PEER_LOCALMSPID="Org2MSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
  export CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
  export CORE_PEER_ADDRESS=peer0.org2.example.com:9051
}

createChannel() {
  echo ">>> [1/4] 创建通道 ${CHANNEL_NAME}..."
  setGlobalsOrg1
  
  BLOCK_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/channel-config/${CHANNEL_NAME}.block"
  if [ ! -f "$BLOCK_FILE" ]; then
    configtxgen -profile medchannel -outputBlock "$BLOCK_FILE" -channelID ${CHANNEL_NAME} -configPath /opt/gopath/src/github.com/hyperledger/fabric/peer/channel-config
  fi

  echo ">>> [2/4] Orderer 加入通道..."
  osnadmin channel join --channelID ${CHANNEL_NAME} --config-block "$BLOCK_FILE" -o orderer.example.com:7053 || true


  echo ">>> [3/4] Peer0.Org1 (Hospital A) 加入通道..."
  setGlobalsOrg1
  peer channel join -b "$BLOCK_FILE" || true

  echo ">>> [4/4] Peer0.Org2 (Hospital B) 加入通道..."
  setGlobalsOrg2
  peer channel join -b "$BLOCK_FILE" || true

  echo ">>> 通道 ${CHANNEL_NAME} 配置成功！"
}


deployChaincode() {
  echo ">>> [Chaincode] 开始打包并部署医疗智能合约 ${CC_NAME}..."
  
  setGlobalsOrg1
  COMMITTED_SEQ=$(peer lifecycle chaincode querycommitted -C ${CHANNEL_NAME} --name ${CC_NAME} 2>/dev/null | grep -o 'Sequence: [0-9]*' | awk '{print $2}' || true)
  if [ -n "$COMMITTED_SEQ" ]; then
    CC_SEQUENCE=$((COMMITTED_SEQ + 1))
    echo ">>> 检测到链上已有已提交版本 (Sequence: ${COMMITTED_SEQ})，自动升级至 Sequence: ${CC_SEQUENCE}..."
  fi

  peer lifecycle chaincode package ${CC_NAME}.tar.gz --path /opt/gopath/src/github.com/chaincode --lang golang --label ${CC_NAME}_${CC_VERSION}
  
  echo ">>> 在 Org1 (Hospital A) 安装链码..."
  peer lifecycle chaincode install ${CC_NAME}.tar.gz || true
  
  echo ">>> 在 Org2 (Hospital B) 安装链码..."
  setGlobalsOrg2
  peer lifecycle chaincode install ${CC_NAME}.tar.gz || true

  PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid ${CC_NAME}.tar.gz)
  echo ">>> 链码 Package ID: ${PACKAGE_ID}"

  echo ">>> Org1 审批链码定义..."
  setGlobalsOrg1
  peer lifecycle chaincode approveformyorg -o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --package-id ${PACKAGE_ID} --sequence ${CC_SEQUENCE}

  echo ">>> Org2 审批链码定义..."
  setGlobalsOrg2
  peer lifecycle chaincode approveformyorg -o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --package-id ${PACKAGE_ID} --sequence ${CC_SEQUENCE}

  echo ">>> 向 medchannel 提交智能合约定义..."
  setGlobalsOrg1
  peer lifecycle chaincode commit -o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --sequence ${CC_SEQUENCE} --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles $PEER0_ORG1_CA --peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles $PEER0_ORG2_CA

  echo ">>> 智能合约 ${CC_NAME} 部署成功！"
}

case "$1" in
  "channel")
    createChannel
    ;;
  "deploy")
    deployChaincode
    ;;
  "all")
    createChannel
    deployChaincode
    ;;
  *)
    echo "Usage: network.sh {channel|deploy|all}"
    exit 1
    ;;
esac
