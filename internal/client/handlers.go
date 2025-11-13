package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
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

func InitUpload(apiURL, filePath string) (*UploadPlan, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	reqBody := map[string]interface{}{
		"filename": fileInfo.Name(),
		"size":     fileInfo.Size(),
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

		// Upload chunk concurrently
		wg.Add(1)
		go func(target UploadTarget, chunkData []byte) {
			defer wg.Done()
			err := uploadChunk(target.URL, chunkData)
			if err != nil {
				fmt.Printf("❌ Chunk %d failed: %v\n", target.ChunkIndex, err)
			} else {
				fmt.Printf("✅ Chunk %d uploaded to %s\n", target.ChunkIndex, target.Node)
			}
		}(target, chunkData)
	}
	wg.Wait()
	return nil
}

func uploadChunk(url string, chunkData []byte) error {
	req, err := http.NewRequest("PUT", url, bytes.NewReader(chunkData))
	if err != nil {
		return err
	}

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

	// 4. Download chunks sequentially
	for _, t := range targets {
		resp, err := http.Get(t.URL)
		if err != nil {
			return fmt.Errorf("failed to download chunk: %v", err)
		}
		_, err = io.Copy(outFile, resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to copy chunk data: %v", err)
		}
		fmt.Printf("✅ Chunk %d downloaded from %s\n", t.ChunkIndex, t.URL)
	}

	fmt.Printf("✅ File downloaded to %s\n", outputPath)
	return nil
}
