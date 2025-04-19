package main

import (
	"strings"
	"errors"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type AtomicIDGenerator struct {
	lastTime     int64
	counter      int64
	randomSuffix string
}

// This random ID thing is a lot harder than I thought
func NewAtomicIDGenerator() *AtomicIDGenerator {

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)

	return &AtomicIDGenerator{
		randomSuffix: hex.EncodeToString(randBytes),
	}
}

func (g *AtomicIDGenerator) Generate() string {
	now := time.Now().UnixMilli()

	last := atomic.LoadInt64(&g.lastTime)

	if now == last {
		count := atomic.AddInt64(&g.counter, 1)
		return fmt.Sprintf("%016x-%04x-%s", now, count, g.randomSuffix)
	}

	atomic.StoreInt64(&g.counter, 0)
	atomic.StoreInt64(&g.lastTime, now)

	return fmt.Sprintf("%016x-%04x-%s", now, 0, g.randomSuffix)
}

func basicPathSanitize(pathName string) (string, error) {
	cleanPath := filepath.Join("./static", pathName)

	log.Print(cleanPath)
	if !strings.HasPrefix(cleanPath, "static") {
		return "", errors.New("invalid path")
	}

	return cleanPath, nil
}

/*
This is a handler function it satisfies the definition of the http.uploadHandler
Implicit interfaces! Wow go is cool! The interface is shown below

	type Handler interface {
	 ServeHTTP(ResponseWriter, *Request)
	}
*/
func uploadHandler(generator *AtomicIDGenerator) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

		const maxMemory = 50 * 1024 * 1024

		request.ParseMultipartForm(maxMemory)

		file, header, error := request.FormFile("file")
		if error != nil {
			http.Error(writer, "Error getting file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		uniqueID := generator.Generate()

		pathName := filepath.Join("uploads", (header.Filename + "."+uniqueID))

		pathName, error = basicPathSanitize(pathName)
		if error != nil {
			http.Error(writer, "Ah you wily fox! Caught you!", http.StatusTeapot)
		}

		/*
			What if the filename already exists? Create some method of handling this for bonus points?

			I did it above! Addendum I actually scrapped the original solution which is now below

			if _, error := os.Stat(pathName); error ==nil{
				filepath.Join("_copy",pathName)
			}

			Actually I am going to solve two problems in one go by appending a unique ID to each file
			This is so if two users upload a file with the same name at the same time
			I don't want to run into a race condition.
			Appending unique ids also solves the race condition issue without resorting to mutexes
			mutual exclusion locks introduce performance overhead and are complicated
			Two birds with one stone!

			Addendum 2: actually turns out trying to do this without a mutex was a lot more complicated
			than I original anticipated. I had to look into atomic operations, closures and the &(variable)
			operator. I'll be honest this was a quite the rabbit hole to go down all to avoid importing
			some package that does safe concurrent ID generation but I learned a lot. I'm not entirely
			confident that my implementation handles all concurrency edgecases but I think this works?

			Closures: Enclosed methods that can access a variable in the scope they are enclosed into

			func doSomething(var something) var otherSomething return{
				func doSomethingElseButUsingSomething return val {
					something = "wow I can access something!"
				}
			}

			atomic operations: I'll be honest I don't really understand these that well.
			That's why I lack full confidence in this implementation.
			My basic understanding is that an atomic opration is a way you can interact with a variable
			from multiple threads IE shared state between threads but the operations are mutually exclusive
			preventing race conditions. This is done through some low level hardware operations
			that are beyond my understanding as of now. I couldn't find a great explanation to reference.
		*/

		destination, error := os.Create(pathName)
		if error != nil {
			http.Error(writer, "Error creating file"+error.Error(), http.StatusInternalServerError)
		}
		defer destination.Close()

		// Copy to file if no errors are thrown
		if _, error := io.Copy(destination, file); error != nil {
			http.Error(writer, "Error saving file", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(writer, "File %s uploaded successfully", header.Filename)
	}
}

func main() {

	idGenerator := NewAtomicIDGenerator()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	fmt.Println("Server started on Port 8080")

	log.Println("FileServer started on :8080")

	//Passing uploadHandler function! this is possible by satisfying http.Handler interface!
	//Yay functional programming ideas!
	http.HandleFunc("/file", uploadHandler(idGenerator))

	//Spawn go routines to handle incoming requests with ListenAndServe.
	//Concurrency by default!
	//wow concurrency here turned out to be a lot more complicated than I thought
	log.Fatal(http.ListenAndServe(":8080", nil))

}
