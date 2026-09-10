package main

import (
	"bufio"
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

	/* Output =
	FileName =  example.txt
	FileSize =  12
	File Modified Time Last Time =  2026-09-10 10:13:01.5497977 +0530 IST &{32 {3635673320 31277278} {3697597689 31277278} {3697597689 31277278} 0 12}
	File or Folder =  false
	File Permissions =  -rw-rw-rw-
	*/

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

	/* Output =
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

	// We don't use ReadFile everytime because it load all content of that file once at a time in memory.
	// If file is small then it is OK but of large file then that file occupy more space in menory and that create a problem for application as well as machine.
	f2, err := os.ReadFile("example.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println("Read file data by ReadFile Method = ", string(f2))

	// Output = Read file data by ReadFile Method =  Hello GOLANG

	// 2.3 What if we have Big File How we Read it.
	// For that we use STREAMING method in GOLANG
	// We discuss this method in future.
	//  ****************************************

	// 3 Folder Read ****************************************
	// dir, err := os.Open(".")
	// this is current directory
	dir, err := os.Open("../")
	// this is one previous directory
	if err != nil {
		panic(err)
	}

	defer dir.Close()

	folderInfo, err := dir.ReadDir(-1)

	for _, fi := range folderInfo {
		fmt.Println(fi.Name(), fi.IsDir())
	}

	// 4. Create a File ****************************************

	f4, err := os.Create("example2.txt")
	if err != nil {
		panic(err)
	}
	defer f4.Close()

	f4.WriteString("Hi this is GOLANG file")

	// If we run the same line f4.WriteString("Hi this is GOLANG file") then it bydefault APPEND means add then new item into the older one.

	f4.WriteString("This is my new line")

	// 4.1 Now How to replace the new Text with another ****************************************

	// 5 Now How to transfer one file data to another by using STREAMING method ****************************************
	// Baiscally Transfer data from eample to example2

	sourceFile, err := os.Open("example.txt")
	if err != nil {
		panic(err)
	}

	defer sourceFile.Close()

	destFile, err := os.Create("example3.txt")
	if err != nil {
		panic(err)
	}

	defer destFile.Close()

	// For Using STREAMING FASHION, We have a inbuild package bufIo

	reader := bufio.NewReader(sourceFile)
	writer := bufio.NewWriter(destFile)

	for {
		b, err := reader.ReadByte()

		if err != nil {
			if err.Error() != "EOF" {
				panic(err)
			}
			break
		}

		error := writer.WriteByte(b)
		if err != nil {
			panic(error)
		}
	}

	// At end after this loop, If any data left we flush the data.
	// So we use writer.Flush()

	writer.Flush()

	fmt.Println("Writting to new file Succesfully.")

	// 6. Now How to Copy one file data to another file ****************************************

	// 7. How to Delete a file

	// er := os.Remove("example2.txt")
	// if er != nil {
	// 	panic(er)
	// }
	// fmt.Println("File Deleted Succesfully...")

}
