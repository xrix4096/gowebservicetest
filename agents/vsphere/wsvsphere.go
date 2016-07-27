package main

import (
	"fmt"
	"runtime"
	"flag"
	"net/url"
	"golang.org/x/net/context"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
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
	fmt.Printf("Root folder: '%v'\n",myClient.ServiceContent.RootFolder)
	fmt.Printf("About: '%+v'\n",myClient.ServiceContent.About)

//	myPropCollector := myClient.PropertyCollector()


	//
	// Optionally dump some JSON info of the client
	//
	if true == *jsonFlag {
		myJSON, _ := myClient.MarshalJSON()
		fmt.Printf("CP: JSON '%s'\n", myJSON)
	}

	//
	// Create a 'finder'
	//
	myFinder := find.NewFinder(myClient.Client, true)

	//
	// List the datacenters at the specified path
	//
	myDCs, err := myFinder.DatacenterList(ctx, *datacenterPath)
	fmt.Printf("Got '%d' datacenter objects\n", len(myDCs))

	for _, element := range myDCs {
		myDatacenter := element
		dumpDatacenterInfo(ctx, myDatacenter)
	}

}

//
// dumpDatacenterInfo:
//  Dump interesting info about a Datacenter object
//
func dumpDatacenterInfo(ctx context.Context, thisDC *object.Datacenter) {
	fmt.Printf("\nDatacenter\n")
	fmt.Printf("----------\n\n")
	fmt.Printf("Name: \t\t\t\t %v (%v)\n", thisDC.Name(), thisDC.Reference())
	fmt.Printf("InventoryPath: \t\t\t %v\n", thisDC.InventoryPath)

	myFolders, err := thisDC.Folders(ctx)
	if err != nil {
		fmt.Printf("Failed to get folders for datacenter: %s\n", err)
		return
	}

	fmt.Printf("Virtual Machine Folder: \t %v\n",
		myFolders.VmFolder.InventoryPath)
	fmt.Printf("Host Folder: \t\t\t %v\n", myFolders.HostFolder.InventoryPath)
	fmt.Printf("Datastore Folder: \t\t %v\n",
		myFolders.DatastoreFolder.InventoryPath)
	fmt.Printf("Network Folder: \t\t %v\n",
		myFolders.NetworkFolder.InventoryPath)


}
