package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var chunks map[int][]byte
var chunksMutex sync.Mutex

func uploadChunk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow access from all origins
	// Limit maximum upload size to 100 MB
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	r.ParseMultipartForm(10 << 20) // Set maximum file size (10 MB in this case)

	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()

	chunkIndex, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		fmt.Println("Error Parsing Chunk Index")
		fmt.Println(err)
		return
	}

	chunk, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error Reading Chunk")
		fmt.Println(err)
		return
	}

	chunksMutex.Lock()
	if chunkIndex == 0 {

	}
	if chunks == nil {
		chunks = make(map[int][]byte)
	}
	chunks[chunkIndex] = chunk
	chunksMutex.Unlock()

	fmt.Printf("Uploaded Chunk: %d\n", chunkIndex)

	fmt.Fprintf(w, "Chunk Uploaded Successfully")
}

func mergeChunks(filename string) error {
	if filename == "" {
		return fmt.Errorf("original file name is not set")
	}
	//Line 66 - 89 will reconstruct chunk
	// Membuat file sementara untuk menyimpan hasil rekonstruksi
	tempFileName := "reconstructed_" + filename
	tempFile, err := os.Create(tempFileName)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	// Menulis chunk ke dalam file sementara sesuai urutan
	chunksMutex.Lock()
	for i := 0; ; i++ {
		chunk, ok := chunks[i]
		if !ok {
			break
		}
		_, err = tempFile.Write(chunk)
		if err != nil {
			chunksMutex.Unlock()
			return err
		}
		delete(chunks, i) // Opsional, hapus chunk setelah digunakan
	}
	chunksMutex.Unlock()

	// It will zip file that already reconstruct (it can remove if file that uploaded already zip)
	// Membuat file .zip
	zipFileName := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".zip" // Mengoreksi penamaan
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// Menambahkan file yang telah direkonstruksi ke dalam .zip
	fileInZip, err := zw.Create(filename) // Mempertahankan nama dan ekstensi file asli
	if err != nil {
		return err
	}
	// Membaca file sementara
	tempFile.Seek(0, 0) // Kembali ke awal file
	_, err = io.Copy(fileInZip, tempFile)
	if err != nil {
		return err
	}

	// Opsional: Menghapus file sementara setelah digunakan
	/*
		err = os.Remove(tempFileName)
		if err != nil {
			return err
		}*/

	return nil
}

func main() {
	http.HandleFunc("/upload", uploadChunk)
	http.HandleFunc("/merge", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		name := r.URL.Query().Get("filename")
		if name == "" {
			http.Error(w, "Filename is required", http.StatusBadRequest)
			return
		}

		err := mergeChunks(name)
		if err != nil {
			fmt.Println("Error Merging Chunks")
			fmt.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		fmt.Println("File Merged Successfully")
		fmt.Fprintf(w, "File Merged Successfully")
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
