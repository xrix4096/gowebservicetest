package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	analyticsModeInvalid       = 0
	analyticsModeListBuckets   = 100
	analyticsModeGetBucketInfo = 200
)

type analyticsResponse struct {
	Name       string
	DateStamp  time.Time
	BucketList []bucketDescription
}

type bucketDescription struct {
	Name         string
	CreationDate *time.Time
}

//
// main
//
func main() {
	fmt.Printf("Initialising Boshing Sequence on %s....\n",
		runtime.GOOS)
	http.HandleFunc("/", requestHandler)
	http.ListenAndServe(":8080", nil)
}

func getModeFromURL(u string) (mode int, bucketID string) {
	mode = analyticsModeInvalid

	fmt.Printf("CP: getModeFromURL '%s'\n", u)
	urlSplit := strings.Split(u, "/")
	count := len(urlSplit)
	fmt.Printf("CP: Have '%d' components in path\n", count)

	for index, element := range urlSplit {
		fmt.Printf("CP: %d -> '%s'\n", index, element)
	}

	if urlSplit[1] != "s3" || urlSplit[2] != "buckets" {
		return analyticsModeInvalid, ""
	}

	if 3 == count {
		mode = analyticsModeListBuckets
		fmt.Printf("List buckets mode\n")
	}
	if 4 == count {
		mode = analyticsModeGetBucketInfo
		fmt.Printf("Bucket info mode\n")
		bucketID = urlSplit[3]
	}

	return mode, bucketID
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("bucketsRequestHandler:")
	fmt.Printf("URL: %s\n", r.URL.String())
	fmt.Printf("METHOD: %s\n", r.Method)

	//
	// Parse the URL to see what type of request we are making. To be valid
	// the first part of the path must be "s3". Following that we may have
	// "/s3/buckets" to request a bucket list, or "/s3/buckets/<bucketid>"
	// to request info on a particular bucket
	//
	analyticsMode, bucketID := getModeFromURL(r.URL.Path)
	fmt.Printf("CP: Mode '%d' Bucket '%s'\n", analyticsMode, bucketID)

	//
	// Supported Modes:
	//  analyticsModeListBuckets
	//  analyticsModeGetBucketInfo
	//
	switch analyticsMode {
	case analyticsModeListBuckets:
		listBuckets(w, r)
	case analyticsModeGetBucketInfo:
		getBucketInfo(w, r, bucketID)
	case analyticsModeInvalid:
		http.Error(w, "Unsupported path requested", 404)
	}

}

func listBuckets(w http.ResponseWriter, r *http.Request) {
	//
	// Only supported method is GET
	//
	if r.Method != "GET" {
		http.Error(w, "Invalid request method", 405)
	}

	//
	// Read query parameters
	//
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Failed to parse form\n")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//
	// "fields" parameter is an array
	//
	values := r.Form["fields"]
	fmt.Printf("Request specifies '%d' fields\n", len(values))

	if 0 < len(values) {
		for index := 0; index < len(values); index++ {
			fmt.Printf("Field %d: '%s'\n", index, values[index])
		}
	}

	//
	// Define response structure
	//
	myResponse := analyticsResponse{"ListBuckets",
		time.Now(),
		make([]bucketDescription, 0)}

	//
	// Do some interacting with the S3 API
	//
	listS3Buckets(&myResponse)

	//
	// Add response headers
	//
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	//

	// Send analyticsResponse as JSON
	//
	responseJSON, err := json.Marshal(myResponse)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(responseJSON)

}

func listS3Buckets(response *analyticsResponse) {

	//
	// @todo Find out how to specify credentials here rather than from global
	// config
	//
	log.Printf("Creating session....")
	mySession := session.New(&aws.Config{Region: aws.String("us-west-2")})
	log.Printf("Connecting to S3....")
	myS3svc := s3.New(mySession)

	log.Printf("Listing buckets....")
	result, err := myS3svc.ListBuckets(&s3.ListBucketsInput{})

	if err != nil {
		log.Println("Failed to list buckets", err)
		return
	}

	log.Println("Buckets:")
	for _, bucket := range result.Buckets {
		log.Printf("%s : %s\n", aws.StringValue(bucket.Name), bucket.CreationDate)
		myBucket := bucketDescription{*bucket.Name, bucket.CreationDate}
		response.BucketList = append(response.BucketList, myBucket)

	}
}

func listS3ObjectsInBucket(response *analyticsResponse, bucketID string) {

	//
	// @todo Find out how to specify credentials here rather than from global
	// config
	//
	log.Printf("Creating session....")
	mySession := session.New(&aws.Config{Region: aws.String("us-east-1")})
	log.Printf("Connecting to S3....")
	myS3svc := s3.New(mySession)

	log.Printf("Listing objects in '%s'....", bucketID)

	i := 0
	err := myS3svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: &bucketID,
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		fmt.Println("Page,", i)
		i++

		for _, obj := range p.Contents {
			fmt.Println("Object:", *obj.Key)
		}
		return true
	})

	if err != nil {
		fmt.Println("Failed to list objects\n", err)
	}
}

func getBucketInfo(w http.ResponseWriter, r *http.Request, bucketID string) {
	//
	// Only supported method is GET
	//
	if r.Method != "GET" {
		http.Error(w, "Invalid request method", 405)
	}

	//
	// Read query parameters
	//
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Failed to parse form\n")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//
	// "fields" parameter is an array
	//
	values := r.Form["fields"]
	fmt.Printf("Request specifies '%d' fields\n", len(values))

	if 0 < len(values) {
		for index := 0; index < len(values); index++ {
			fmt.Printf("Field %d: '%s'\n", index, values[index])
		}
	}

	//
	// Define response structure
	//
	myResponse := analyticsResponse{"BucketInfo",
		time.Now(),
		make([]bucketDescription, 0)}

	//
	// Do some interacting with the S3 API
	//
	listS3ObjectsInBucket(&myResponse, bucketID)

	//
	// Add response headers
	//
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	//

	// Send analyticsResponse as JSON
	//
	responseJSON, err := json.Marshal(myResponse)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(responseJSON)

}
