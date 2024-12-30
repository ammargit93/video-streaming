package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"golang.org/x/crypto/bcrypt"
)

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateThumbnail(savedFilePath string, saveFileName string) error {
	ffmpegPath := "C:/Users/Ammar1/Downloads/ffmpeg-master-latest-win64-gpl/ffmpeg-master-latest-win64-gpl/bin/ffmpeg"
	cmd := exec.Command(ffmpegPath, "-i", savedFilePath, "-ss", "00:00:01.000", "-vframes", "1", saveFileName)

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
	return nil
}

// Read and save thumbnail into local storage.
func ReadAndSaveThumbnail(ctx *gin.Context, file *multipart.FileHeader) (string, string) {
	savePath := filepath.Join("C:/Users/Ammar1/go/video-streaming/videos/", file.Filename)
	imgPath := filepath.Join("C:/Users/Ammar1/go/video-streaming/static/images/", file.Filename[:len(file.Filename)-3]+"png")
	if err := ctx.SaveUploadedFile(file, savePath); err != nil {
		log.Println("Error saving uploaded file:", err)
		ctx.JSON(500, gin.H{"Error": "Failed to save uploaded file"})

	}
	if err := GenerateThumbnail(savePath, imgPath); err != nil {
		ctx.JSON(500, gin.H{"Error": "Error generating thumbnail"})

	}

	return file.Filename[:len(file.Filename)-3] + "png", ""
}

func UploadToGridFS(fileHeader *multipart.FileHeader, db *mongo.Database) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	bucket, err := gridfs.NewBucket(db) // Replace `db` with your MongoDB database instance
	if err != nil {
		return "", err
	}
	uploadStream, err := bucket.OpenUploadStream(fileHeader.Filename)
	if err != nil {
		return "", err
	}
	defer uploadStream.Close()
	if _, err := io.Copy(uploadStream, file); err != nil {
		return "", err
	}

	return uploadStream.FileID.(primitive.ObjectID).Hex(), nil
}

func GetFromGridFS(fileID string, db *mongo.Database) ([]byte, error) {
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return nil, err
	}

	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = bucket.DownloadToStream(objectID, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// func Base64ToImage(base64String, outputPath string) error {
// 	if strings.Contains(base64String, ",") {
// 		parts := strings.Split(base64String, ",")
// 		base64String = parts[1]
// 	}
// 	imageData, err := base64.StdEncoding.DecodeString(base64String)
// 	if err != nil {
// 		return fmt.Errorf("failed to decode Base64 string: %w", err)
// 	}
// 	file, err := os.Create(outputPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to create image file: %w", err)
// 	}
// 	defer file.Close()
// 	_, err = file.Write(imageData)
// 	if err != nil {
// 		return fmt.Errorf("failed to write image data to file: %w", err)
// 	}

// 	return nil
// }
