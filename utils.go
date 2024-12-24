package main

import (
	"bytes"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateThumbnail(savedFilePath string) error {
	ffmpegPath := "C:/Users/Ammar1/Downloads/ffmpeg-master-latest-win64-gpl/ffmpeg-master-latest-win64-gpl/bin/ffmpeg"
	cmd := exec.Command(ffmpegPath, "-i", savedFilePath, "-ss", "00:00:01.000", "-vframes", "1", "thumbnail.jpg")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error generating thumbnail:", err)
		fmt.Println("FFmpeg stderr:", stderr.String())
		return err
	}
	fmt.Println("FFmpeg output:", out.String())
	return nil
}

func ReadAndSaveThumbnail(ctx *gin.Context, file *multipart.FileHeader) []byte {
	savePath := filepath.Join("C:/Users/Ammar1/go/video-streaming/videos/", file.Filename)
	if err := ctx.SaveUploadedFile(file, savePath); err != nil {
		log.Println("Error saving uploaded file:", err)
		ctx.JSON(500, gin.H{"Error": "Failed to save uploaded file"})

	}
	if err := GenerateThumbnail(savePath); err != nil {
		ctx.JSON(500, gin.H{"Error": "Error generating thumbnail"})

	}
	thumbnail, err := os.ReadFile("thumbnail.jpg")
	if err != nil {
		log.Println("Error reading thumbnail:", err)
		ctx.JSON(500, gin.H{"Error": "Error reading thumbnail file"})
	}
	return thumbnail
}
