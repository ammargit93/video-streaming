package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	client          *mongo.Client
	videoCollection *mongo.Collection
	usersCollection *mongo.Collection
	store           = sessions.NewCookieStore([]byte("secret-key"))
)

type User struct {
	Userid      string  `json:"userid" bson:"userid"`
	Username    string  `json:"username" bson:"username"`
	Password    string  `json:"password" bson:"password"`
	Email       string  `json:"email" bson:"email"`
	Age         int     `json:"age" bson:"age"`
	Nationality string  `json:"nationality" bson:"nationality"`
	Videos      []Video `json:"videos" bson:"videos"`
}

type Video struct {
	Videoid     string        `json:"videoid" bson:"videoid"`
	Videotitle  string        `json:"videotitle" bson:"videotitle"`
	Videodesc   string        `json:"videodesc" bson:"videodesc"`
	Videolength time.Time     `json:"videolength" bson:"videolength"`
	Comments    []interface{} `json:"comments" bson:"comments"`
}

var err error

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

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
	videoCollection = client.Database("streamdb").Collection("videos")
	usersCollection = client.Database("streamdb").Collection("users")

	fmt.Println("HELLO")

	r := gin.Default()
	r.LoadHTMLGlob("templates/*.html")

	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(200, "home.html", nil)
	})
	r.GET("/signup", func(ctx *gin.Context) {
		ctx.HTML(200, "signup.html", nil)
	})
	r.GET("/login", func(ctx *gin.Context) {
		ctx.HTML(200, "login.html", nil)
	})

	r.POST("/signup", func(ctx *gin.Context) {
		username := ctx.PostForm("username")
		password := ctx.PostForm("password")
		email := ctx.PostForm("email")
		age := ctx.PostForm("age")
		nationality := ctx.PostForm("nationality")

		hashed, _ := bcrypt.GenerateFromPassword([]byte(password), 8)

		var user User
		user.Userid = uuid.New().String()
		user.Username = username
		user.Password = string(hashed)
		user.Email = email
		user.Age, _ = strconv.Atoi(age)
		user.Nationality = nationality

		var potentialUser User
		err = usersCollection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&potentialUser)
		if err != nil {
			_, err = usersCollection.InsertOne(ctx, user)
			if err != nil {
				log.Println("Error inserting a user.")
			}
		} else {
			fmt.Sprintf(`<script>alert("Username already exists, try a different name.")</script>`)
			ctx.Redirect(http.StatusFound, "signup")
		}

		ctx.Redirect(http.StatusFound, "login")
	})

	r.POST("/login", func(ctx *gin.Context) {
		username := ctx.PostForm("username")
		password := ctx.PostForm("password")

		var user User
		usersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
		result := CheckPasswordHash(password, user.Password)

		if !result {
			fmt.Println("Wrong Password Try again")
		} else {
			session, err := store.Get(ctx.Request, "curr-session")
			if err != nil {
				fmt.Println("Error declaring or retrieving a sesssion")
			}
			session.Values["username"] = username
			session.Values["password"] = password
			session.Values["email"] = user.Email
			session.Values["age"] = user.Age
			session.Values["nationality"] = user.Nationality

			ctx.Redirect(http.StatusFound, "")

		}
	})

	r.Run(":8080")
}
