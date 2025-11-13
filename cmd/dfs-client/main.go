package main

import (
	"fmt"
	"os"

	"github.com/timskillet/distributed-filestore/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("Upload: dfs-client upload <API_SERVER_URL> <FILE_PATH>")
		fmt.Println("Download: dfs-client download <API_SERVER_URL> <FILE_ID> <OUTPUT_PATH>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "upload":
		if len(os.Args) < 4 {
			fmt.Println("Usage: dfs-client upload <API_SERVER_URL> <FILE_PATH>")
			os.Exit(1)
		}
		apiURL := os.Args[2]
		filePath := os.Args[3]
		fmt.Println("Initializing upload for:", filePath)

		// 1. Initialize upload plan
		uploadPlan, err := client.InitUpload(apiURL, filePath)
		if err != nil {
			panic(err)
		}

		// 2. Upload chunks concurrently
		fmt.Printf("Uploading file %s in %dMB chunks...\n", filePath, uploadPlan.ChunkSize/(1024*1024))
		err = client.UploadChunks(filePath, uploadPlan)
		if err != nil {
			panic(err)
		}

		// 3. Finalize upload
		err = client.FinalizeUpload(apiURL, uploadPlan.FileID)
		if err != nil {
			panic(err)
		}

		fmt.Println("✅ Upload complete:", uploadPlan.FileID)

	case "download":
		if len(os.Args) != 5 {
			fmt.Println("Usage: dfs-client download <API_SERVER_URL> <FILE_ID> <OUTPUT_PATH>")
			os.Exit(1)
		}
		apiURL := os.Args[2]
		fileID := os.Args[3]
		outputPath := os.Args[4]

		if err := client.DownloadFile(apiURL, fileID, outputPath); err != nil {
			panic(err)
		}

	default:
		fmt.Println("Invalid command:", command)
	}
}
