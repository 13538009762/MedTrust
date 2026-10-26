package ipfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IPFSService IPFS 统一服务接口 (支持 Live 与 Mock 降级互换)
type IPFSService interface {
	Add(cipherBytes []byte) (cid string, err error)
	Cat(cid string) (cipherBytes []byte, err error)
	PutData(ciphertext []byte) (string, error)
	GetData(cid string) ([]byte, error)
}

// KuboLiveClient 通过 Kubo HTTP API 连接真实 Docker 容器或守护节点的实现
type KuboLiveClient struct {
	apiURL     string
	storageDir string
	client     *http.Client
	mu         sync.RWMutex
}

type ipfsAddResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

func NewKuboLiveClient(apiURL, storageDir string) *KuboLiveClient {
	_ = os.MkdirAll(storageDir, 0755)
	return &KuboLiveClient{
		apiURL:     apiURL,
		storageDir: storageDir,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *KuboLiveClient) Add(cipherBytes []byte) (string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "encrypted_record.bin")
	if err == nil {
		_, _ = fw.Write(cipherBytes)
		_ = w.Close()

		req, reqErr := http.NewRequest("POST", c.apiURL+"/api/v0/add", &b)
		if reqErr == nil {
			req.Header.Set("Content-Type", w.FormDataContentType())
			resp, doErr := c.client.Do(req)
			if doErr == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					bodyBytes, readErr := io.ReadAll(resp.Body)
					if readErr == nil {
						var addResp ipfsAddResponse
						if jErr := json.Unmarshal(bodyBytes, &addResp); jErr == nil && addResp.Hash != "" {
							realCID := addResp.Hash
							localPath := filepath.Join(c.storageDir, realCID)
							_ = os.WriteFile(localPath, cipherBytes, 0644)
							log.Printf("[IPFS-Live] 成功写入真实 IPFS 节点，CID: %s", realCID)
							return realCID, nil
						}
					}
				} else {
					log.Printf("[IPFS-Live] 节点返回非 200 状态码 (%d)，自动降级", resp.StatusCode)
				}
			}
		}
	}

	// 若远程调用失败则平滑降级写入本地适配器
	log.Printf("[IPFS-Live] 节点连接异常，自动降级本地存储: %s", c.apiURL)
	return (&LocalMockIPFSClient{storageDir: c.storageDir}).Add(cipherBytes)
}

func (c *KuboLiveClient) Cat(cid string) ([]byte, error) {
	localPath := filepath.Join(c.storageDir, cid)
	if data, err := os.ReadFile(localPath); err == nil && len(data) > 0 {
		return data, nil
	}

	reqURL := fmt.Sprintf("%s/api/v0/cat?arg=%s", c.apiURL, cid)
	resp, err := c.client.Post(reqURL, "", nil)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			remoteData, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(remoteData) > 0 {
				_ = os.WriteFile(localPath, remoteData, 0644)
				return remoteData, nil
			}
		}
	}

	return (&LocalMockIPFSClient{storageDir: c.storageDir}).Cat(cid)
}

func (c *KuboLiveClient) PutData(ciphertext []byte) (string, error) {
	return c.Add(ciphertext)
}

func (c *KuboLiveClient) GetData(cid string) ([]byte, error) {
	return c.Cat(cid)
}

// LocalMockIPFSClient 本地降级 Mock 实现 (对密文计算哈希生成 Qm CID，并写入本地磁盘)
type LocalMockIPFSClient struct {
	storageDir string
	mu         sync.RWMutex
}

func NewLocalMockIPFSClient(storageDir string) *LocalMockIPFSClient {
	_ = os.MkdirAll(storageDir, 0755)
	return &LocalMockIPFSClient{
		storageDir: storageDir,
	}
}

func (m *LocalMockIPFSClient) Add(cipherBytes []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := sha256.Sum256(cipherBytes)
	cid := fmt.Sprintf("Qm%s%s", hex.EncodeToString(hash[:16]), hex.EncodeToString(hash[16:32]))

	localPath := filepath.Join(m.storageDir, cid)
	if err := os.WriteFile(localPath, cipherBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to persist to local mock IPFS: %w", err)
	}

	log.Printf("[IPFS-Mock] 密文存入本地模拟 IPFS，CID: %s", cid)
	return cid, nil
}

func (m *LocalMockIPFSClient) Cat(cid string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	localPath := filepath.Join(m.storageDir, cid)
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("mock IPFS file not found: %s", cid)
	}
	return data, nil
}

func (m *LocalMockIPFSClient) PutData(ciphertext []byte) (string, error) {
	return m.Add(ciphertext)
}

func (m *LocalMockIPFSClient) GetData(cid string) ([]byte, error) {
	return m.Cat(cid)
}

// CreateIPFSService 工厂函数
func CreateIPFSService(mode, apiURL, storageDir string) IPFSService {
	_ = os.MkdirAll(storageDir, 0755)
	if mode == "live" {
		return NewKuboLiveClient(apiURL, storageDir)
	}
	return NewLocalMockIPFSClient(storageDir)
}

// NewIPFSService 向后兼容旧版初始化构造函数
func NewIPFSService(apiURL, storageDir string) IPFSService {
	return CreateIPFSService("live", apiURL, storageDir)
}
