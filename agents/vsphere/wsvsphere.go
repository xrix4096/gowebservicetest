package main

import (
	"fmt"
	"runtime"
	"flag"
	"path"
	"net/url"
	"golang.org/x/net/context"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

//
// Globals
//


//
// Command line flags
//
var userFlag = flag.String("user", "default", "ESX / vCenter user")
var pwFlag = flag.String("password", "", "ESX / vCenter password")
var urlFlag = flag.String("url", "https://username:password@host/sdk",
	"ESX / vCenter URL")
var datacenterPath = flag.String("dcpath", "", "Path containing datacenter(s)")
var verboseFlag = flag.Bool("verbose", false,
	"Extra verbose output")


//
// main
//
func main() {
	//
	// Process CLI arguments
	//
	flag.Parse()

	if *verboseFlag {
		fmt.Printf("Running on %s....\n", runtime.GOOS)
	}

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

	if true == *verboseFlag {
		fmt.Printf("DEBUG: FullURL = '%s'\n", u.String())
	}

	//
	// Try and connect to that vCenter
	//
	myClient, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	//
	// Print service information
	//
	myAboutInfo := myClient.ServiceContent.About
	fmt.Printf("Server Name: \t\t\t %v\n", myAboutInfo.FullName)
	fmt.Printf("API Type: \t\t\t %v\n", myAboutInfo.ApiType)
	fmt.Printf("API Version: \t\t\t %v\n", myAboutInfo.ApiVersion)
	fmt.Printf("Root folder: \t\t\t %v\n",myClient.ServiceContent.RootFolder)
	fmt.Printf("InstanceID: \t\t\t %v\n",myAboutInfo.InstanceUuid)
	fmt.Printf("Product: \t\t\t %v\n", myAboutInfo.LicenseProductName)
	fmt.Printf("Product Version: \t\t %v\n", myAboutInfo.LicenseProductVersion)
	fmt.Printf("Product Line: \t\t\t %v\n", myAboutInfo.ProductLineId)
	fmt.Printf("Host OS: \t\t\t %v\n", myAboutInfo.OsType)
	fmt.Printf("Vendor: \t\t\t %v\n", myAboutInfo.Vendor)

	//
	// Optionally dump some JSON info of the client
	//
	if true == *verboseFlag {
		myJSON, _ := myClient.MarshalJSON()
		fmt.Printf("CP: JSON '%s'\n", myJSON)
	}

	//
	// Create a 'finder'
	//
	myFinder := find.NewFinder(myClient.Client, true)

	//
	// @todo Use finder.ManagedObjectList() to walk from "/" and find any
	// datacenters rather than needing to specify the DC path on the CLI
	//

	//
	// List the datacenters at the specified path
	//
	myDCs, err := myFinder.DatacenterList(ctx, *datacenterPath)
	fmt.Printf("\nGot '%d' datacenter objects\n", len(myDCs))

	for _, element := range myDCs {
		myDatacenter := element
		dumpDatacenterInfo(ctx, myClient, myFinder, myDatacenter)
	}


}

//
// dumpDatacenterInfo:
//  Dump interesting info about an object.Datacenter object
//
func dumpDatacenterInfo(ctx context.Context,
	myClient *govmomi.Client,
	myFinder *find.Finder,
	thisDC *object.Datacenter) {
	fmt.Printf("\nDatacenter\n")
	fmt.Printf("----------\n\n")

	//
	// Basic Datacenter info
	//
	fmt.Printf("Name: \t\t\t\t %v (%v)\n", thisDC.Name(), thisDC.Reference())
	fmt.Printf("InventoryPath: \t\t\t %v\n", thisDC.InventoryPath)

	//
	// Get the various folders
	//
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

	//
	// Get host system list
	//
	myHosts, err := myFinder.HostSystemList(ctx,
		path.Join(myFolders.HostFolder.InventoryPath, "*"))
	if err != nil {
		fmt.Printf("Failed to get host system list: %v\n", err)
		return
	}

	fmt.Printf("%d Hosts:\n\n", len(myHosts))
	for index, element := range myHosts {
		fmt.Printf("-- Host %d --\n", index)
		myHost := element
		dumpHostSystemInfo(ctx, myFinder, myHost)
		fmt.Print("\n")
	}

	//
	// Get datastore list
	//
	myDatastores, err := myFinder.DatastoreList(ctx,
		path.Join(myFolders.DatastoreFolder.InventoryPath, "*"))
	if err != nil {
		fmt.Printf("Failed to get datastore list: %v\n", err)
		return
	}

	//
	// Fetch full Datastore info via a PropertyCollector
	//
	// filterProps specifies which fields we'd like returned
	//
	filterProps := []string{"info", "summary"}

	//
	// myResult structure passed to and from property collector
	//
	var myResult infoResult

	//
	// Put list of Datastores in myResult.objects
	//
	myResult.objects = append(myResult.objects, myDatastores...)
	if len(myResult.objects) > 0 {
		//
		// Make a set of object references for each of the datastores
		//
		dsRefs := make([]types.ManagedObjectReference, 0, len(myResult.objects))
		for _, element := range myResult.objects {
			dsRefs = append(dsRefs, element.Reference())
		}

		//
		// Run Retrieve() on the property collector
		//
		myPC := property.DefaultCollector(myClient.Client)
		err := myPC.Retrieve(ctx, dsRefs, filterProps, &myResult.Datastores)
		if err != nil {
			fmt.Printf("Failed to run PC: %v\n", err)
			return
		}

		//
		// myResult.Datastores now contains the requested datastore info. Since
		// the property collector does not guarantee the order of the results
		// matches the order of the list of requested references we make a map
		// that allows us to access them my their ManagedObjectReference and
		// walk the source list (myRequest.objects) in that order pulling
		// in info from the map by ManagedObjectReference as needed
		//
		fullDatastores := make(map[types.ManagedObjectReference]mo.Datastore,
			len(myResult.Datastores))
		for _, element := range myResult.Datastores {
			fullDatastores[element.Reference()] = element
		}

		//
		// Now we have all the info; the basic object.Datastore info in
		// myResult.objects and the mo.Datastore info in the fullDatastores map
		//
		for index, element := range myResult.objects {
			fmt.Printf("-- Datastore %d --\n", index)
			refDS := element
			fullDS := fullDatastores[refDS.Reference()]
			dumpDatastoreInfo(ctx, myFinder, refDS, &fullDS)
			fmt.Print("\n")
		}
	}
}


//
// infoResult:
//  Type used by property collector to return results
//
type infoResult struct {
	Datastores []mo.Datastore
	objects    []*object.Datastore
}


//
// dumpHostSystemInfo:
//  Dump interesting info about an object.HostSystem object
//
func dumpHostSystemInfo(ctx context.Context,
	myFinder *find.Finder,
	thisHost *object.HostSystem) {

	//
	// Basic host system info
	//
	fmt.Printf("Name: \t\t\t\t %v (%v)\n",
		thisHost.Name(),
		thisHost.Reference())
	fmt.Printf("InventoryPath: \t\t\t %v\n",
		thisHost.InventoryPath)

	//
	// Management IP address(es)
	//
	myIPs, err := thisHost.ManagementIPs(ctx)
	if err != nil {
		fmt.Printf("Failed to get IPs for host: %v\n", err)
		return
	}

	fmt.Printf("Management IPs: \t\t ")
	for _, element := range myIPs {
		thisIP := element
		fmt.Print(thisIP)
	}
	fmt.Printf("\n")

	//
	// Resource Pool
	//
	myResPool, err := thisHost.ResourcePool(ctx)
	if err != nil {
		fmt.Printf("Failed to get resource pool for host: %v", err)
		return
	}

	resPoolName := myResPool.Name()
	if resPoolName == "" {
		resPoolName = "(Unnamed)"
	}
	fmt.Printf("Resource Pool: \t\t\t %v (%v)\n",
		resPoolName,
		myResPool.Reference())
}

//
// dumpDatastoreInfo:
//  Dump interesting info about an object.Datastore object
//
func dumpDatastoreInfo(ctx context.Context,
	myFinder *find.Finder,
	thisDatastore *object.Datastore,
	fullDatastore *mo.Datastore) {

	dsSummary := fullDatastore.Summary

	//
	// Basic datastore info
	//
	fmt.Printf("Name: \t\t\t\t %v (%v)\n",
		thisDatastore.Name(),
		thisDatastore.Reference())
	fmt.Printf("InventoryPath: \t\t\t %v\n",
		thisDatastore.InventoryPath)
	fmt.Printf("Type: \t\t\t\t %v\n", dsSummary.Type)
	fmt.Printf("URL: \t\t\t\t %v\n", dsSummary.Url)
	fmt.Printf("Capacity: \t\t\t %v\n", dsSummary.Capacity)
	fmt.Printf("Free Space: \t\t\t %v\n", dsSummary.FreeSpace)
	fmt.Printf("MultiHost: \t\t\t %v\n", *dsSummary.MultipleHostAccess)
}

//
// printTypeAndValue:
//  Debug function, quickly check on type and value of an unknown typed var
//
func printTypeAndValue(name string, myVar interface {}) {
	fmt.Printf("CP: %s: '%T' '%+v'\n", name, myVar, myVar)
}
