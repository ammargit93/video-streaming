package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"golang.org/x/crypto/bcrypt"
)

const (
	ffmpegPath = "C:/Users/Ammar1/Downloads/ffmpeg-master-latest-win64-gpl/ffmpeg-master-latest-win64-gpl/bin/ffmpeg"
	// Name of the file in the bucket
)

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateThumbnail(savedFilePath string, imageName string) error {
	cmd := exec.Command(ffmpegPath, "-i", savedFilePath, "-ss", "00:00:01.000", "-vframes", "1", imageName)

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
	// os.Remove("thumbnail.png")
	return nil
}

func ReadAndSaveThumbnail(ctx *gin.Context, file *multipart.FileHeader) (string, string) {
	var key = "images/"
	bucket := "aws-video-streaming-image-bucket"

	savePath := filepath.Join("C:/Users/Ammar1/go/video-streaming/videos/", file.Filename)
	imgPath := file.Filename[:len(file.Filename)-3] + "png"

	if err := ctx.SaveUploadedFile(file, savePath); err != nil {
		log.Println("Error saving uploaded file:", err)
		ctx.JSON(500, gin.H{"Error": "Failed to save uploaded file"})

	}
	if err := GenerateThumbnail(savePath, imgPath); err != nil {
		ctx.JSON(500, gin.H{"Error": "Error generating thumbnail"})
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"))
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	newfile, err := os.Open(file.Filename[:len(file.Filename)-3] + "png")
	if err != nil {
		log.Fatalf("failed to open file %q: %v", file.Filename[:len(file.Filename)-3]+"png", err)
	}
	defer newfile.Close()

	key += file.Filename[:len(file.Filename)-3] + "png"
	input := &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        newfile,
		ContentType: aws.String("image/png"), // Update based on your file type
	}
	_, err = client.PutObject(context.TODO(), input)
	if err != nil {
		log.Fatalf("failed to upload file: %v", err)
	}

	fmt.Printf("Successfully uploaded %q to bucket %q\n", key, bucket)
	os.Remove(savePath)

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

func VideoExists(ctx *gin.Context, video Video) bool {
	var existingVideo Video
	err := videoCollection.FindOne(ctx, bson.M{
		"videoauthor": video.Videoauthor,
		"videotitle":  video.Videotitle,
		"videosize":   video.Videosize,
	}).Decode(&existingVideo)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false
		}
		log.Println("Error checking for duplicate video:", err)
		ctx.JSON(500, gin.H{"Error": "Failed to check for duplicate video"})
	}

	return true
}
