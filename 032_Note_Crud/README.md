<!-- Step 1 -->
go mod init notes_api

<!-- Now we need to install some dependency -->
<!-- To install some package in Go Project Similar lile we install in Node like npm i and package name  -->

go get github.com/gin-gonic/gin@latest

go get github.com/joho/godotenv 

go get go.mongodb.org/mongo-driver/mongo

go get go.mongodb.org/mongo-driver/mongo/options


<!-- For Automatic Reflect the change we have air@latest -->

go install github.com/air-verse/air@latest

<!-- Then ENV code -->
MONGO_URI=mongodb+srv://alokkumarcse01_db_user:9IUhje69bKP91u1h@cluster0.kltgpx0.mongodb.net/
MONGO_DB_NAME=notes_db
PORT=8080

Now we creata a db/mongo.go and inside that we creata two functions one is Connect() and another is disconnect.

Now we make a router by using GIN.