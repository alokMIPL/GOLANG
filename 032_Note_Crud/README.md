<!-- Step 1 -->
go mod init notes_api

<!-- Now we need to install some dependency -->
<!-- To install some package in Go Project Similar lile we install in Node like npm i and package name  -->

go get github.com/gin-gonic/gin@latest

<!-- Then ENV code -->
MONGO_URI=mongodb+srv://alokkumarcse01_db_user:9IUhje69bKP91u1h@cluster0.kltgpx0.mongodb.net/
MONGO_DB_NAME=notes_db
PORT=8080