package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client          *mongo.Client
	videoCollection *mongo.Collection
	usersCollection *mongo.Collection
	store           = sessions.NewCookieStore([]byte("secret-key"))
	err             error
)

func main() {
	var clientOptions = options.Client().ApplyURI("mongodb://localhost:27017")
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	var db = client.Database("streamdb")

	videoCollection = client.Database("streamdb").Collection("videos")
	usersCollection = client.Database("streamdb").Collection("users")

	r := gin.Default()
	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")

	r.GET("/", func(ctx *gin.Context) {
		var videos []Video
		cursor, err := videoCollection.Find(ctx, bson.M{})
		if err != nil {
			log.Println("Error finding videos:", err)
			ctx.JSON(500, gin.H{"Error": "Failed to retrieve videos"})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &videos); err != nil {
			log.Println("Error decoding videos:", err)
			ctx.JSON(500, gin.H{"Error": "Failed to decode video data"})
			return
		}
		ctx.HTML(200, "home.html", gin.H{"videos": videos})
	})

	r.GET("/signup", func(ctx *gin.Context) {
		ctx.HTML(200, "signup.html", nil)
	})
	r.GET("/login", func(ctx *gin.Context) {
		ctx.HTML(200, "login.html", nil)
	})

	r.POST("/signup", SignupPOST)

	r.POST("/login", LoginPOST)

	r.GET("/logout", Logout)

	r.GET("/profile", func(ctx *gin.Context) {
		session, err := store.Get(ctx.Request, "curr-session")
		if err != nil {
			http.Error(ctx.Writer, "Failed to get session", http.StatusInternalServerError)
			return
		}
		cursor, err := videoCollection.Find(ctx, bson.M{"videoauthor": session.Values["username"].(string)})
		if err != nil {
			log.Println("Error finding videos:", err)
			ctx.JSON(500, gin.H{"Error": "Failed to retrieve videos"})
			return
		}
		defer cursor.Close(ctx)
		var userVideos []Video
		if err := cursor.All(ctx, &userVideos); err != nil {
			log.Println("Error decoding videos:", err)
			ctx.JSON(500, gin.H{"Error": "Failed to decode video data"})
			return
		}

		ctx.HTML(200, "profile.html", gin.H{"username": session.Values["username"].(string), "videos": userVideos})
	})

	r.POST("/profile", func(ctx *gin.Context) {
		file, err := ctx.FormFile("video")
		if err != nil {
			log.Println("Error reading formfile:", err)
			ctx.JSON(400, gin.H{"Error": "Error reading video file"})
			return
		}
		session, err := store.Get(ctx.Request, "curr-session")
		if err != nil {
			http.Error(ctx.Writer, "Failed to get session", http.StatusInternalServerError)
			return
		}
		thumbnail, _ := ReadAndSaveThumbnail(ctx, file)
		fileID, _ := UploadToGridFS(file, client.Database("streamdb"))
		var video Video = Video{
			Videoid:        uuid.New().String(),
			Videoauthor:    session.Values["username"].(string),
			Videotitle:     file.Filename,
			Videodesc:      ctx.PostForm("video_description"),
			Videosize:      file.Size,
			Videofileid:    fileID,
			Videothumbnail: thumbnail,
		}

		if file.Header.Get("Content-Type") != "video/mp4" {
			ctx.JSON(400, gin.H{"Error": "Only video/mp4 files allowed"})
			return
		}
		_, err = videoCollection.InsertOne(ctx, video)
		if err != nil {
			log.Println("Error inserting video:", err)
			ctx.JSON(500, gin.H{"Error": "Video insertion failed"})
			return
		}

		var user User
		user.Videos = append(user.Videos, video)
		_, err = usersCollection.UpdateOne(ctx, bson.M{"username": session.Values["username"].(string)}, bson.M{
			"$set": bson.M{"videos": user.Videos},
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var userVideos []Video

		cursor, err := videoCollection.Find(ctx, bson.M{"videoauthor": session.Values["username"].(string)})
		if err != nil {
			log.Println("Error finding videos:", err)
			ctx.JSON(500, gin.H{"Error": "Failed to retrieve videos"})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &userVideos); err != nil {
			log.Println("Error decoding videos:", err)
			ctx.JSON(500, gin.H{"Error": "Failed to decode video data"})
			return
		}

		ctx.HTML(200, "profile.html", gin.H{"username": session.Values["username"].(string), "videos": userVideos})
	})

	r.GET("/watch/:video_id", func(ctx *gin.Context) {
		videoid := ctx.Param("video_id")
		var videoToPlay Video
		videoCollection.FindOne(ctx, bson.M{"videoid": videoid}).Decode(&videoToPlay)
		// videoBytes, _ := GetFromGridFS(videoToPlay.Videofileid, db)

		// ctx.Header("Content-Type", "video/mp4") // Set MIME type
		// ctx.Header("Accept-Ranges", "bytes")    // Enable range requests
		// ctx.Writer.WriteHeader(200)             // HTTP 200 OK
		// ctx.Writer.Write(videoBytes)
		ctx.HTML(200, "basevideoplayer.html", gin.H{
			"videotitle":  videoToPlay.Videotitle,
			"videodesc":   videoToPlay.Videodesc,
			"videoid":     videoToPlay.Videofileid,
			"videoauthor": videoToPlay.Videoauthor,
		})

	})

	r.GET("/video/:video_id", func(ctx *gin.Context) {
		videoid := ctx.Param("video_id")
		var videoToPlay Video
		videoCollection.FindOne(ctx, bson.M{"videoid": videoid}).Decode(&videoToPlay)
		videoBytes, _ := GetFromGridFS(videoid, db)

		ctx.Header("Content-Type", "video/mp4") // Set MIME type
		ctx.Header("Accept-Ranges", "bytes")    // Enable range requests
		ctx.Writer.WriteHeader(200)             // HTTP 200 OK
		ctx.Writer.Write(videoBytes)
	})

	fmt.Println("Starting the Server.")
	r.Run(":8080")
}
