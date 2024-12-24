package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func SignupPOST(ctx *gin.Context) {
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
		ctx.Redirect(http.StatusFound, "signup")
	}
	ctx.Redirect(http.StatusFound, "login")
}

func LoginPOST(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")

	var user User
	usersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	result := CheckPasswordHash(password, user.Password)

	if !result {
		ctx.JSON(200, gin.H{"error": "Wrong Password Try again"})
	} else {
		session, err := store.Get(ctx.Request, "curr-session")
		if err != nil {
			ctx.JSON(200, gin.H{"error": "Error declaring or retrieving a sesssion"})
		}
		session.Values["username"] = username
		session.Values["password"] = password
		session.Values["email"] = user.Email
		session.Values["age"] = user.Age
		session.Values["nationality"] = user.Nationality
		session.Save(ctx.Request, ctx.Writer)

		ctx.Redirect(http.StatusFound, "")

	}
}

func Logout(ctx *gin.Context) {
	session, err := store.Get(ctx.Request, "curr-session")
	if err != nil {
		http.Error(ctx.Writer, "Failed to get session", http.StatusInternalServerError)
		return
	}
	session.Values = map[interface{}]interface{}{}
	session.Options.MaxAge = -1
	err = session.Save(ctx.Request, ctx.Writer)
	if err != nil {
		http.Error(ctx.Writer, "Failed to clear session", http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusFound, "")
}
