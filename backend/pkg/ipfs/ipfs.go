package ipfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type IPFSService struct {
	apiURL     string
	storageDir string
	client     *http.Client
	mu         sync.RWMutex
}

func NewIPFSService(apiURL, storageDir string) *IPFSService {
	_ = os.MkdirAll(storageDir, 0755)
	return &IPFSService{
		apiURL:     apiURL,
		storageDir: storageDir,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *IPFSService) PutData(ciphertext []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256(ciphertext)
	cid := fmt.Sprintf("Qm%s%s", hex.EncodeToString(hash[:16]), hex.EncodeToString(hash[16:32]))

	localPath := filepath.Join(s.storageDir, cid)
	if err := os.WriteFile(localPath, ciphertext, 0644); err != nil {
		return "", fmt.Errorf("failed to persist to IPFS storage: %w", err)
	}

	go func() {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("file", cid)
		if err == nil {
			fw.Write(ciphertext)
			w.Close()
			req, err := http.NewRequest("POST", s.apiURL+"/api/v0/add", &b)
			if err == nil {
				req.Header.Set("Content-Type", w.FormDataContentType())
				resp, err := s.client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}()

	return cid, nil
}

func (s *IPFSService) GetData(cid string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	localPath := filepath.Join(s.storageDir, cid)
	if data, err := os.ReadFile(localPath); err == nil {
		return data, nil
	}

	reqURL := fmt.Sprintf("%s/api/v0/cat?arg=%s", s.apiURL, cid)
	resp, err := s.client.Post(reqURL, "", nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("IPFS CID %s not found", cid)
}
