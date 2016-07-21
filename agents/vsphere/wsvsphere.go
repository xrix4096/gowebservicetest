package main

import (
	"fmt"
	"runtime"
	"net/url"
	"golang.org/x/net/context"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
)


//
// main
//
func main() {
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
	// @todo: Fetch following variables from CLI args or local config
	//
	myUser := "backup_user@vSphere.local"
	myPW := "Binars1!"
	myBaseURL := "https://ukpvmvcd05.dsgdev.lab/sdk"

	//
	// Create full URL from the credentials, base and path
	//
	u, err := url.Parse(myBaseURL)
	if err != nil {
		fmt.Printf("Failed to parse base URL '%s': '%s'\n", myBaseURL, err)
		return
	}
	fmt.Printf("CP: URL = '%s'\n", u.String())

	u.User = url.UserPassword(myUser, myPW)
	fmt.Printf("CP: URL = '%s'\n", u.String())

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
	myJSON, err := myClient.MarshalJSON()
	fmt.Printf("CP: JSON '%s'\n", myJSON)

	//
	// Create a 'finder'
	//
	myFinder := find.NewFinder(myClient.Client, true)

	//
	// Find the default datacenter
	//
	myDefaultDC, err := myFinder.Datacenter(ctx, "/Poole/NetVault Development")
	if err != nil {
		fmt.Println(err)
		return
	}
	myFinder.SetDatacenter(myDefaultDC)

	myObjName, err := myDefaultDC.ObjectName(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("CP: ObjName '%s'\n", myObjName)


}
