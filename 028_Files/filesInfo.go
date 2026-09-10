package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.Open("example.txt")
	// The os.Open() return two things.
	// It return two things.
	// 1. File-Object which is pointer
	// 2. error

	// Now i go when error comes we need to handle it or return it, we can't thow that error in GOLANG
	// So,
	if err != nil {
		// Log the error

		// Panic the Program
		panic(err)
	}

	// We handle the error Now file_Object turn
	// In file we have lots of options to use like
	/*
		f.Stat()
		f.Name()
		f.Read()
		f.ReadAt()
	*/

	/*
		Now take f.Stat(), same it takes os.FileInfo, and error.
	*/

	fileInfo, err := f.Stat()
	// again same for this also.
	// Now i go when error comes we need to handle it or return it, we can't thow that error in GOLANG
	// So,
	if err != nil {
		// Log the error

		// Panic the Program
		panic(err)
	}

	// Same with fileInfo, we have various methods with file info like
	/*
		fileInfo.Name()
		fileInfo.Mode()
		fileInfo.Size()
	*/

	fmt.Println("FileName = ", fileInfo.Name())
	fmt.Println("FileSize = ", fileInfo.Size())
	fmt.Println("File Modified Time Last Time = ", fileInfo.ModTime())
	fmt.Println("", fileInfo.Sys())
	fmt.Println("File or Folder = ", fileInfo.IsDir())
	fmt.Println("File Permissions = ", fileInfo.Mode())

}
