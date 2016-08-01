package main

import (
	"fmt"
	"runtime"
	"flag"
	"path"
	"net/url"
	"encoding/json"
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
	// Get root folder
	//
	rootFolder, err := getFolderFromRef(ctx,
		myClient,
		myClient.ServiceContent.RootFolder)
	if err != nil {
		fmt.Printf("Failed to get root folder: %v\n", err)
		return
	}

	//
	// Print service information
	//
	myAboutInfo := myClient.ServiceContent.About
	fmt.Printf("Server Name: \t\t\t %v\n", myAboutInfo.FullName)
	fmt.Printf("API Type: \t\t\t %v\n", myAboutInfo.ApiType)
	fmt.Printf("API Version: \t\t\t %v\n", myAboutInfo.ApiVersion)
	fmt.Printf("Root folder: \t\t\t %v (%v)\n",
		rootFolder.Name,
		rootFolder.Self)
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
	// Fetch full Datacenter info via property collector
	//
	var myDCInfo mo.Datacenter

	//
	// Run RetrieveOne() on the  property collector
	//
	myPC := property.DefaultCollector(myClient.Client)
	err := myPC.RetrieveOne(ctx, thisDC.Reference(), nil, &myDCInfo)
	if err != nil {
		fmt.Printf("Failed to get DC info from property collector: %s", err)
		return
	}

	//
	// Basic Datacenter info
	//
	fmt.Printf("Name: \t\t\t\t %v (%v)\n", thisDC.Name(), thisDC.Reference())
	fmt.Printf("Status: \t\t\t %v\n", myDCInfo.OverallStatus)
	fmt.Printf("InventoryPath: \t\t\t %v\n", thisDC.InventoryPath)

	//
	// Get parent folder
	//
	parentInfo, err := getFolderFromRef(ctx, myClient, *myDCInfo.Parent)
	if err != nil {
		fmt.Printf("Failed to get parent folder: %v\n", err)
		return
	}
	fmt.Printf("Parent Folder: \t\t\t %s (%v)\n",
		parentInfo.Name,
		parentInfo.Self)

	//
	// Check if the parent folder is at the root. If not we need to prepend
	// the parent prefix to the inventory paths, which appears to be a bug in
	// the API since they really should be returning an absolute value for
	// the InventoryPaths in the datacenter Folders set.
	//
	// @todo Could fix Datacenter.Folders() to set the correct InventoryPath
	//
	var invPrefix = ""
	if *myDCInfo.Parent != myClient.ServiceContent.RootFolder {
		invPrefix = path.Join("/", parentInfo.Name)
	}

	//
	// Get the various subfolders
	//
	myFolders, err := thisDC.Folders(ctx)
	if err != nil {
		fmt.Printf("Failed to get folders for datacenter: %s\n", err)
		return
	}

	fmt.Printf("Virtual Machine Folder: \t %v (%v)\n",
		myFolders.VmFolder.InventoryPath,
		myDCInfo.VmFolder)
	fmt.Printf("Host Folder: \t\t\t %v (%v)\n",
		myFolders.HostFolder.InventoryPath,
		myDCInfo.HostFolder)
	fmt.Printf("Datastore Folder: \t\t %v (%v)\n",
		myFolders.DatastoreFolder.InventoryPath,
		myDCInfo.DatastoreFolder)
	fmt.Printf("Network Folder: \t\t %v (%v)\n",
		myFolders.NetworkFolder.InventoryPath,
		myDCInfo.NetworkFolder)

	//
	// Get host system list
	//
	myHosts, err := myFinder.HostSystemList(ctx,
		path.Join(invPrefix, myFolders.HostFolder.InventoryPath, "*"))
	if err != nil {
		fmt.Printf("Failed to get host system list: %v\n", err)
		return
	}

	fmt.Printf("%d Hosts:\n\n", len(myHosts))
	for index, element := range myHosts {
		fmt.Printf("-- Host %d --\n", index)
		myHost := element
		dumpHostSystemInfo(ctx, myClient, myFinder, myHost)
		fmt.Print("\n")
	}

	//
	// Get datastore list
	//
	myDatastores, err := myFinder.DatastoreList(ctx,
		path.Join(invPrefix, myFolders.DatastoreFolder.InventoryPath, "*"))
	if err != nil {
		fmt.Printf("Failed to get datastore list: %v\n", err)
		return
	}

	//
	// Fetch full Datastore info via a PropertyCollector. This is a fair
	// example of how to be efficient by fetching a whole bunch of objects
	// at the same time, as opposed to the RetrieveOne() approach that is
	// simpler but results in many requests
	//
	// filterProps specifies which fields we'd like returned
	//
	filterProps := []string{
		"info",
		"summary",
		"capability",
		"parent",
		"overallStatus",
	}

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
	myClient *govmomi.Client,
	myFinder *find.Finder,
	thisHost *object.HostSystem) {

	var props []string
	props = []string {
		"hardware",
		"overallStatus",
		"summary",
	}

	var fullHost mo.HostSystem

	//
	// Get serverside properties
	//
	err := thisHost.Properties(ctx, thisHost.Reference(), props, &fullHost)
	if err != nil {
		fmt.Printf("Failed to get serverside properties of host: '%v'\n", err)
		return
	}

	//
	// Basic host system info
	//
	fmt.Printf("Name: \t\t\t\t %v (%v)\n",
		thisHost.Name(),
		thisHost.Reference())
	fmt.Printf("InventoryPath: \t\t\t %v\n",
		thisHost.InventoryPath)
	fmt.Printf("Status: \t\t\t %v\n", fullHost.OverallStatus)
	fmt.Printf("Model Type: \t\t\t %v - %v\n",
		fullHost.Summary.Hardware.Vendor,
		fullHost.Summary.Hardware.Model)
	fmt.Printf("Memory: \t\t\t %v\n", fullHost.Summary.Hardware.MemorySize)
	fmt.Printf("CPU Type: \t\t\t %v\n", fullHost.Summary.Hardware.CpuModel)
	fmt.Printf("CPU Clock: \t\t\t %v\n", fullHost.Summary.Hardware.CpuMhz)
	fmt.Printf("Cores x Threads: \t\t %v x %v\n",
		fullHost.Summary.Hardware.NumCpuCores,
		fullHost.Summary.Hardware.NumCpuThreads)
	fmt.Printf("Power: \t\t\t\t %v\n", fullHost.Summary.Runtime.PowerState)
	fmt.Printf("Boot Time: \t\t\t %v\n", fullHost.Summary.Runtime.BootTime)
	fmt.Printf("Maintenance Mode: \t\t %v\n",
		fullHost.Summary.Runtime.InMaintenanceMode)
	fmt.Printf("Product: \t\t\t %v\n",
		fullHost.Summary.Config.Product.FullName)
	fmt.Printf("Management Server IP \t\t %v\n",
		fullHost.Summary.ManagementServerIp)


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
	dsCaps := fullDatastore.Capability
	vmfsInfo := fullDatastore.Info.GetDatastoreInfo()

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
	fmt.Printf("Status: \t\t\t %v\n", fullDatastore.OverallStatus)
	fmt.Printf("Parent: \t\t\t %v\n", fullDatastore.Parent)
	fmt.Printf("Supports Directory Hierarchy: \t %v\n",
		dsCaps.DirectoryHierarchySupported)
	fmt.Printf("Native Snapshots: \t\t %v\n", *dsCaps.NativeSnapshotSupported)
	fmt.Printf("Per-file Thin Provisioning: \t %v\n",
		dsCaps.PerFileThinProvisioningSupported)
	fmt.Printf("Supports Raw Disk Mappings: \t %v\n",
		dsCaps.RawDiskMappingsSupported)
	fmt.Printf("Supports StorageIORM: \t\t %v\n", *dsCaps.StorageIORMSupported)
	fmt.Printf("Sparse Files: \t\t\t %v\n", *dsCaps.SeSparseSupported)
	fmt.Printf("Max File Size: \t\t\t %v\n", vmfsInfo.MaxFileSize)
	fmt.Printf("Max Virtual Disk Size: \t\t %v\n",
		vmfsInfo.MaxVirtualDiskCapacity)
	fmt.Printf("Max Mem File Size: \t\t %v\n", vmfsInfo.MaxMemoryFileSize)
	fmt.Printf("Timestamp: \t\t\t %v\n", vmfsInfo.Timestamp)
}


func getFolderFromRef(ctx context.Context,
	myClient *govmomi.Client,
	myRef types.ManagedObjectReference) (mo.Folder, error) {

	var myResult mo.Folder
	myPC := property.DefaultCollector(myClient.Client)
	err := myPC.RetrieveOne(ctx, myRef, nil, &myResult)

	return myResult, err
}


//
// dumpFolderInfo:
//  Print out folder information
//
func dumpFolderInfo(ctx context.Context,
	myClient *govmomi.Client,
	myRef types.ManagedObjectReference) {

	var myFolderInfo mo.Folder

	myPC := property.DefaultCollector(myClient.Client)
	err := myPC.RetrieveOne(ctx, myRef, nil, &myFolderInfo)
	if err != nil {
		fmt.Printf("Failed to get folder '%s': '%s'\n", myRef, err)
		return
	}

	fmt.Printf("%v (%v)", myFolderInfo.Name, myFolderInfo.Self)
	//	printAsJSON("CP FOLDER INFO DUMP", myFolderInfo)
}


//
// printTypeAndValue:
//  Debug function, quickly check on type and value of an unknown typed var
//
func printTypeAndValue(name string, myVar interface {}) {
	fmt.Printf("CP: %s: '%T' '%+v'\n", name, myVar, myVar)
}

//
// printAsJSON:
//  Debug function, dump out the specified variable in JSON format
//
func printAsJSON(name string, myVar interface {}) {
	myJSON, _ := json.MarshalIndent(myVar, "", "    ")
	fmt.Printf("CP: %s: '%s'\n", name, myJSON)
}
