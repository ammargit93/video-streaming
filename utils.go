package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"log"
	"mime/multipart"

	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const (
	ffmpegPath = "C:/Users/Ammar1/Downloads/ffmpeg-master-latest-win64-gpl/ffmpeg-master-latest-win64-gpl/bin/ffmpeg"
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

	// Upload the thumbnail to S3
	newfile, err := os.Open(imgPath)
	if err != nil {
		log.Fatalf("failed to open file %q: %v", imgPath, err)
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

	fmt.Printf("Successfully uploaded thumbnail %q to bucket %q\n", key, bucket)

	// Remove the local video and thumbnail file after upload
	// os.Remove(savePath)
	os.Remove(imgPath)

	return imgPath, ""
}

var redisClient = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379", // Redis server address
	Password: "",               // No password set
	DB:       0,                // Default DB
})

func GetVideoFromS3(id string, ctx *gin.Context) {

	cachedVideo, err := redisClient.Get("video:" + id).Result()
	if err == redis.Nil {
		var video Video
		if err := videoCollection.FindOne(ctx, bson.M{"videoid": id}).Decode(&video); err != nil {
			log.Printf("Failed to find video in database: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video data"})
			return
		}

		cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"))
		bucket := "aws-video-streaming-image-bucket"
		key := "videos/" + video.Videotitle
		client := s3.NewFromConfig(cfg)

		// Fetch video from S3
		input := &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		}
		result, err := client.GetObject(context.TODO(), input)
		if err != nil {
			log.Printf("Failed to get video from S3: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video from S3"})
			return
		}
		defer result.Body.Close()

		// Read the video file from the S3 response body
		videoData, err := ioutil.ReadAll(result.Body)
		if err != nil {
			log.Printf("Failed to read video data: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read video data"})
			return
		}

		err = redisClient.Set("video:"+id, videoData, 60*time.Second).Err()
		if err != nil {
			log.Printf("Failed to cache video in Redis: %v", err)
		}

		// Stream the video to the client
		ctx.Header("Content-Type", "video/mp4")
		ctx.Header("Accept-Ranges", "bytes")
		ctx.Writer.WriteHeader(http.StatusOK)

		_, err = ctx.Writer.Write(videoData)
		if err != nil {
			log.Printf("Error while streaming video: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream video"})
		}
	} else if err != nil {
		// Error occurred while fetching from Redis
		log.Printf("Failed to fetch video from Redis: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video"})
		return
	} else {
		// Video is cached, stream it directly

		ctx.Header("Content-Type", "video/mp4")
		ctx.Header("Accept-Ranges", "bytes")
		ctx.Writer.WriteHeader(http.StatusOK)

		_, err = ctx.Writer.Write([]byte(cachedVideo))
		if err != nil {
			log.Printf("Error while streaming cached video: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream cached video"})
		}
	}
}

func VideoExists(ctx *gin.Context, video Video) bool {
	var existingVideo Video
	err := videoCollection.FindOne(ctx, bson.M{
		"videoauthor": video.Videoauthor,
		"videotitle":  video.Videotitle,
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

func SaveVideoToS3(file *multipart.FileHeader, ctx *gin.Context) {
	key := "videos/"
	bucket := "aws-video-streaming-image-bucket"
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"))
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	newfile, err := os.Open("videos/" + file.Filename)
	if err != nil {
		log.Fatalf("failed to open file %q: %v", file.Filename, err)
	}
	defer newfile.Close()
	key += file.Filename
	input := &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        newfile,
		ContentType: aws.String("video/mp4"),
	}
	_, err = client.PutObject(context.TODO(), input)
	if err != nil {
		log.Fatalf("failed to upload file: %v", err)
	}

	fmt.Printf("Successfully uploaded %q to bucket %q\n", key, bucket)

}
