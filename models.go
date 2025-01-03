package main

import (
	"time"
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
	Videoid        string    `json:"videoid" bson:"videoid"`
	Videoauthor    string    `json:"videoauthor" bson:"videoauthor"`
	Videotitle     string    `json:"videotitle" bson:"videotitle"`
	Videodesc      string    `json:"videodesc" bson:"videodesc"`
	Videocomments  []Comment `json:"videocomments" bson:"videocomments"`
	Videothumbnail any       `json:"videothumbnail" bson:"videothumbnail"`
}

type Comment struct {
	CommentVideoID string    `json:"commentvideoid" bson:"commentvideoid"`
	CommentID      string    `json:"commentid" bson:"commentid"`
	CommentText    string    `json:"commenttext" bson:"commenttext"`
	CommentAuthor  string    `json:"commentauthor" bson:"commentauthor"`
	CommentDate    time.Time `json:"commentdate" bson:"commentdate"`
}

type Reply struct {
	ReplyID       string    `json:"replyid" bson:"replyid"`
	ReplyText     string    `json:"replytext" bson:"replytext"`
	ReplyDate     time.Time `json:"replydate" bson:"replydate"`
	ReplyParentID string    `json:"replyparentid" bson:"replyparentid"` //Foreign key corresponding to Comment.CommentID
}
