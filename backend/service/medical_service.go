package service

import (
	"fmt"
	"medtrust-backend/config"
	"medtrust-backend/pkg/ipfs"
)

type MedicalService struct {
	ipfsService ipfs.IPFSService
}

var DefaultMedicalService *MedicalService

func InitMedicalService(args ...string) {
	mode := "live"
	apiURL := "http://127.0.0.1:5001"
	storageDir := "./data/ipfs_storage"

	if config.AppConfig.IPFS.Mode != "" {
		mode = config.AppConfig.IPFS.Mode
	}
	if config.AppConfig.IPFS.APIURL != "" {
		apiURL = config.AppConfig.IPFS.APIURL
	}
	if config.AppConfig.IPFS.StorageDir != "" {
		storageDir = config.AppConfig.IPFS.StorageDir
	}

	if len(args) == 1 {
		mode = args[0]
	} else if len(args) == 2 {
		apiURL = args[0]
		storageDir = args[1]
	} else if len(args) >= 3 {
		mode = args[0]
		apiURL = args[1]
		storageDir = args[2]
	}

	DefaultMedicalService = &MedicalService{
		ipfsService: ipfs.CreateIPFSService(mode, apiURL, storageDir),
	}
}

func (s *MedicalService) GetIPFSService() ipfs.IPFSService {
	if s == nil || s.ipfsService == nil {
		storageDir := "./data/ipfs_storage"
		apiURL := "http://127.0.0.1:5001"
		mode := "live"
		if config.AppConfig.IPFS.Mode != "" {
			mode = config.AppConfig.IPFS.Mode
		}
		if config.AppConfig.IPFS.APIURL != "" {
			apiURL = config.AppConfig.IPFS.APIURL
		}
		if config.AppConfig.IPFS.StorageDir != "" {
			storageDir = config.AppConfig.IPFS.StorageDir
		}
		return ipfs.CreateIPFSService(mode, apiURL, storageDir)
	}
	return s.ipfsService
}

// GetIPFSData 从 IPFS 节点或本地 IPFS 缓存拉取密文数据
func (s *MedicalService) GetIPFSData(cid string) ([]byte, error) {
	if s.ipfsService == nil {
		return nil, fmt.Errorf("IPFS 服务未初始化")
	}
	return s.ipfsService.GetData(cid)
}
