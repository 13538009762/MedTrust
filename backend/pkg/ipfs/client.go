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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var validCIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{4,128}$`)

func validateCID(cid string) error {
	if !validCIDRe.MatchString(cid) {
		return fmt.Errorf("非法的 IPFS CID 格式")
	}
	return nil
}

// IPFSService IPFS 统一服务接口 (支持 Live 与 Mock 降级互换)
type IPFSService interface {
	Add(cipherBytes []byte) (cid string, err error)
	Cat(cid string) (cipherBytes []byte, err error)
	PutData(ciphertext []byte) (string, error)
	GetData(cid string) ([]byte, error)
	// Delete 清理已写入但后续步骤失败的孤儿密文（DB 事务回滚时调用）。
	// 尽力而为语义：失败不阻断主流程，由调用方记日志。
	Delete(cid string) error
}

// KuboLiveClient 通过 Kubo HTTP API 连接真实 Docker 容器或守护节点的实现
type KuboLiveClient struct {
	apiURL     string
	storageDir string
	client     *http.Client
	fallback   *LocalMockIPFSClient
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
		fallback:   NewLocalMockIPFSClient(storageDir),
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
							if err := validateCID(realCID); err != nil {
								return "", err
							}
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
	return c.fallback.Add(cipherBytes)
}

func (c *KuboLiveClient) Cat(cid string) ([]byte, error) {
	if err := validateCID(cid); err != nil {
		return nil, err
	}
	localPath := filepath.Join(c.storageDir, cid)
	if data, err := os.ReadFile(localPath); err == nil && len(data) > 0 {
		return data, nil
	}

	reqURL := fmt.Sprintf("%s/api/v0/cat?arg=%s", c.apiURL, url.QueryEscape(cid))
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

	return c.fallback.Cat(cid)
}

func (c *KuboLiveClient) PutData(ciphertext []byte) (string, error) {
	return c.Add(ciphertext)
}

func (c *KuboLiveClient) GetData(cid string) ([]byte, error) {
	return c.Cat(cid)
}

// Delete 尽力清理本地缓存副本并尽力通知 Kubo unpin。
// 说明：IPFS 内容寻址存储无法真正"删除"已被其他节点固定的块，此处只做本节点清理，
// 目的是避免本地上传失败留下的孤儿密文长期占用磁盘与被无谓固定。
func (c *KuboLiveClient) Delete(cid string) error {
	if cid == "" {
		return nil
	}
	if err := validateCID(cid); err != nil {
		return err
	}
	localPath := filepath.Join(c.storageDir, cid)
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理本地 IPFS 缓存失败: %w", err)
	}
	// 尽力 unpin，失败不影响主流程（Kubo 不可达时忽略）
	pinURL := fmt.Sprintf("%s/api/v0/pin/rm?arg=%s", c.apiURL, url.QueryEscape(cid))
	if resp, err := c.client.Post(pinURL, "", nil); err == nil {
		_ = resp.Body.Close()
	}
	return nil
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
	if err := validateCID(cid); err != nil {
		return nil, err
	}
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

// Delete 移除本地模拟存储中的密文文件（上传失败后的孤儿清理）
func (m *LocalMockIPFSClient) Delete(cid string) error {
	if cid == "" {
		return nil
	}
	if err := validateCID(cid); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	localPath := filepath.Join(m.storageDir, cid)
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理本地模拟 IPFS 文件失败: %w", err)
	}
	return nil
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
