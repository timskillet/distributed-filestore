package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// --- API Types ---
type UploadTarget struct {
	ChunkIndex int    `json:"chunk_index"`
	Node       string `json:"node"`
	URL        string `json:"url"`
}

type UploadPlan struct {
	FileID        string         `json:"file_id"`
	ChunkSize     int            `json:"chunk_size"`
	UploadTargets []UploadTarget `json:"upload_targets"`
}

const (
	maxRetries        = 3
	initialBackoff    = 1 * time.Second
	backoffMultiplier = 2
)

func InitUpload(apiURL, filePath string, chunkSize int) (*UploadPlan, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	reqBody := map[string]interface{}{
		"filename":   fileInfo.Name(),
		"size":       fileInfo.Size(),
		"chunk_size": chunkSize,
	}

	// Send upload request to API server
	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(apiURL+"/init-upload", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload initialization failed: %s", string(respBody))
	}

	var plan UploadPlan
	err = json.NewDecoder(resp.Body).Decode(&plan)
	if err != nil {
		return nil, fmt.Errorf("failed to decode upload plan: %v", err)
	}
	return &plan, nil
}

func UploadChunks(filePath string, plan *UploadPlan) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedChunks []int
	chunkSize := plan.ChunkSize
	buffer := make([]byte, chunkSize)

	for _, target := range plan.UploadTargets {
		// Read chunk from file
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// Prepare chunk data for upload
		chunkData := make([]byte, n)
		copy(chunkData, buffer[:n])

		// Calculate SHA256 checksum
		hash := sha256.Sum256(chunkData)
		checksum := hex.EncodeToString(hash[:])

		// Upload chunk concurrently with retry
		wg.Add(1)
		go func(target UploadTarget, chunkData []byte, checksum string) {
			defer wg.Done()
			err := uploadChunkWithRetry(target.URL, chunkData, target.ChunkIndex, checksum)
			if err != nil {
				mu.Lock()
				failedChunks = append(failedChunks, target.ChunkIndex)
				mu.Unlock()
				fmt.Printf("❌ Chunk %d failed after retries: %v\n", target.ChunkIndex, err)
			} else {
				fmt.Printf("✅ Chunk %d uploaded to %s (checksum: %s)\n", target.ChunkIndex, target.Node, checksum[:16]+"...")
			}
		}(target, chunkData, checksum)
	}
	wg.Wait()

	if len(failedChunks) > 0 {
		return fmt.Errorf("failed to upload %d chunks: %v", len(failedChunks), failedChunks)
	}
	return nil
}

func uploadChunkWithRetry(url string, chunkData []byte, chunkIndex int, checksum string) error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("⏳ Retrying chunk %d (attempt %d/%d) after %v...\n", chunkIndex, attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
			backoff *= backoffMultiplier
		}

		err := uploadChunk(url, chunkData, checksum)
		if err == nil {
			if attempt > 0 {
				fmt.Printf("✅ Chunk %d succeeded on retry attempt %d\n", chunkIndex, attempt+1)
			}
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("chunk upload failed after %d attempts: %v", maxRetries, lastErr)
}

func uploadChunk(url string, chunkData []byte, checksum string) error {
	req, err := http.NewRequest("PUT", url, bytes.NewReader(chunkData))
	if err != nil {
		return err
	}

	// Include checksum in header for validation
	req.Header.Set("X-Chunk-Checksum", checksum)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chunk upload failed: %s", string(body))
	}
	return nil
}

func FinalizeUpload(apiURL, fileID string) error {
	reqBody := map[string]string{"file_id": fileID}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(apiURL+"/finalize-upload", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("finalized upload failed: %s", string(b))
	}

	fmt.Println("File finalized successfully.")
	return nil
}

type DownloadTarget struct {
	ChunkIndex int    `json:"chunk_index"`
	URL        string `json:"url"`
	Checksum   string `json:"checksum"`
}

func DownloadFile(apiURL, fileID string, outputPath string) error {
	// 1. Get download plan
	resp, err := http.Get(fmt.Sprintf("%s/download-plan?file_id=%s", apiURL, fileID))
	if err != nil {
		return fmt.Errorf("failed to get download plan: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get download plan: %s", string(body))
	}

	var targets []DownloadTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return fmt.Errorf("failed to decode download targets: %v", err)
	}

	// 2. Sort targets by chunk index
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].ChunkIndex < targets[j].ChunkIndex
	})

	// 3. Open output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// 4. Download chunks concurrently
	type chunkResult struct {
		index    int
		data     []byte
		checksum string
		err      error
	}

	var wg sync.WaitGroup
	results := make(chan chunkResult, len(targets))

	for _, t := range targets {
		wg.Add(1)
		go func(t DownloadTarget) {
			defer wg.Done()
			resp, err := http.Get(t.URL)
			if err != nil {
				results <- chunkResult{index: t.ChunkIndex, err: fmt.Errorf("failed to download chunk: %v", err)}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				results <- chunkResult{index: t.ChunkIndex, err: fmt.Errorf("chunk download failed: %s", string(body))}
				return
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- chunkResult{index: t.ChunkIndex, err: fmt.Errorf("failed to read chunk data: %v", err)}
				return
			}

			// Calculate checksum of downloaded chunk
			hash := sha256.Sum256(data)
			calculatedChecksum := hex.EncodeToString(hash[:])

			// Validate checksum if expected checksum is provided
			if t.Checksum != "" && calculatedChecksum != t.Checksum {
				results <- chunkResult{
					index:    t.ChunkIndex,
					err:      fmt.Errorf("checksum mismatch: expected %s, got %s", t.Checksum, calculatedChecksum),
					checksum: calculatedChecksum,
				}
				return
			}

			results <- chunkResult{index: t.ChunkIndex, data: data, checksum: calculatedChecksum}
		}(t)
	}

	// Close results channel when all downloads complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and write in order
	chunkMap := make(map[int][]byte)
	var downloadErrors []error

	for result := range results {
		if result.err != nil {
			fmt.Printf("❌ Chunk %d failed: %v\n", result.index, result.err)
			downloadErrors = append(downloadErrors, result.err)
		} else {
			chunkMap[result.index] = result.data
			if result.checksum != "" {
				fmt.Printf("✅ Chunk %d downloaded and validated (checksum: %s...)\n", result.index, result.checksum[:16])
			} else {
				fmt.Printf("✅ Chunk %d downloaded\n", result.index)
			}
		}
	}

	if len(downloadErrors) > 0 {
		return fmt.Errorf("failed to download %d chunks", len(downloadErrors))
	}

	// Write chunks to file in order
	for i := 0; i < len(targets); i++ {
		chunkData, ok := chunkMap[i]
		if !ok {
			return fmt.Errorf("missing chunk %d", i)
		}
		_, err := outFile.Write(chunkData)
		if err != nil {
			return fmt.Errorf("failed to write chunk %d: %v", i, err)
		}
	}

	fmt.Printf("✅ File downloaded to %s\n", outputPath)
	return nil
}
