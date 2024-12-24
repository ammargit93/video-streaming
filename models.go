package main

import (
	"mime/multipart"
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
	Videoid        string                `json:"videoid" bson:"videoid"`
	Videoauthor    string                `json:"videoauthor" bson:"videoauthor"`
	Videotitle     string                `json:"videotitle" bson:"videotitle"`
	Videodesc      string                `json:"videodesc" bson:"videodesc"`
	Videosize      int64                 `json:"videosize" bson:"videosize"`
	Videofile      *multipart.FileHeader `json:"videofile" bson:"videofile"`
	Videocomments  []interface{}         `json:"videocomments" bson:"videocomments"`
	Videothumbnail []byte                `json:"videothumbnail" bson:"videothumbnail"`
}
