package main

import (
	"fmt"
	"runtime"
	"flag"
	"net/url"
	"golang.org/x/net/context"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
)

//
// Command line flags
//
var userFlag = flag.String("user", "default", "ESX / vCenter user")
var pwFlag = flag.String("password", "", "ESX / vCenter password")
var urlFlag = flag.String("url", "https://username:password@host/sdk",
	"ESX / vCenter URL")
var datacenterPath = flag.String("dcpath", "", "Path containing datacenter(s)")
var jsonFlag = flag.Bool("dumpjson", false,
	"Enable dump of client info as JSON")


//
// main
//
func main() {
	//
	// Process CLI arguments
	//
	flag.Parse()

	fmt.Printf("Running on %s....\n", runtime.GOOS)

	muckAbout()
}

//
// muckAbout:
//  Experiment with govmomi functionality
//
func muckAbout() {
	//
	// Create a copy of the main context with a cancel function to cleanup
	// when this function completes
	//
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()   // Execute on function completion

	//
	// Fetch following variables from CLI args or local config
	//
	myUser := *userFlag
	myPW := *pwFlag
	myBaseURL := *urlFlag

	//
	// Create full URL from the credentials, base and path
	//
	u, err := url.Parse(myBaseURL)
	if err != nil {
		fmt.Printf("Failed to parse base URL '%s': '%s'\n", myBaseURL, err)
		return
	}

	u.User = url.UserPassword(myUser, myPW)
	fmt.Printf("DEBUG: FullURL = '%s'\n", u.String())

	//
	// Try and connect to that vCenter
	//
	fmt.Printf("Connecting.... ")
	myClient, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Connected to server version '%s'\n", myClient.Version)

	//
	// Dump some JSON info of the client
	//
	if true == *jsonFlag {
		myJSON, _ := myClient.MarshalJSON()
		fmt.Printf("CP: JSON '%s'\n", myJSON)
	}

	//
	// Create a 'finder'
	//
	myFinder := find.NewFinder(myClient.Client, true)
	fmt.Printf("CP: Got finder: '%T'\n", myFinder)

	//
	// List the datacenters at the specified path
	//
	myDCs, err := myFinder.DatacenterList(ctx, *datacenterPath)
	fmt.Printf("Got '%d' datacenter objects: '%T'\n",
		len(myDCs), myDCs)

	for _, element := range myDCs {
		myDatacenter := element
		fmt.Printf("Datacenter: '%s'\n", myDatacenter.Name())
		fmt.Printf("DC Type: '%T'\n", myDatacenter)
		fmt.Printf("DC Ref: '%+v'\n", myDatacenter.Reference())
		fmt.Printf("DC Ref Type: '%T'\n", myDatacenter.Reference())
//		dumpDatacenterInfo(myDatacenter)
	}


//	myFinder.SetDatacenter(myDefaultDC)
//	fmt.Printf("Got DC: '%s'\n", reflect.TypeOf(myDefaultDC))

}

//func dumpDatacenterInfo(thisDC *object.Datacenter) {
//	thisDC.arse
//}
