package main

import (
	"fmt"
	"time"
	"net/http"
	"runtime"
	"encoding/json"
	"log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type analyticsResponse struct {
	Name         string
	DateStamp    time.Time
	BucketList []bucketDescription
}

type bucketDescription struct {
	Name          string
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
	if err != nil	{
		fmt.Printf("Failed to parse form\n")
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	myResponse := analyticsResponse{"ANiceResponse", time.Now(), make([]bucketDescription, 0) }

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
