package main

import (
	"fmt"
	"os"
)

func main() {

	// 1. File Information ****************************************
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

	// 2. File Read ****************************************

	// 2.1 File Read by classical way ****************************************

	f1, err := os.Open("example.txt")
	if err != nil {
		panic(err)
	}

	// When we open a file, it must need to be closed.
	defer f1.Close()

	// Now read the file content.
	// So basically when we read any file we need to store that file data into buffer. And buffer is Array of bytes.

	buf := make([]byte, fileInfo.Size())

	d, err := f.Read(buf)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(buf); i++ {
		fmt.Println("Data from file read = ", d, string(buf[i]))
	}

	/*
			Data from file read =  12 H
		Data from file read =  12 e
		Data from file read =  12 l
		Data from file read =  12 l
		Data from file read =  12 o
		Data from file read =  12
		Data from file read =  12 G
		Data from file read =  12 O
		Data from file read =  12 L
		Data from file read =  12 A
		Data from file read =  12 N
		Data from file read =  12 G
	*/

	// 2.2 File Read by ReadFile method ****************************************

	f2, err := os.ReadFile("example.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println("Read file data by ReadFile Method = ", string(f2))

	// Output = Read file data by ReadFile Method =  Hello GOLANG

}
