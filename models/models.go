package models

import (
	"encoding/json"
	"fmt"
	"log"
	"io/ioutil"
	"sync"

	"net/url"

	"github.com/jigsheth57/clone-apps-plugin/apihelper"
)

type Org struct {
	Name        string
	Quota		Quota
	Spaces      Spaces
}

type Space struct {
	Name     				string
	Apps     				Apps
	Services 				Services
	SecurityGroup			SecurityGroups
	StagingSecurityGroup	SecurityGroups
}

//App representation
type App struct {
	Guid                    string
	Name                    string
	Memory                  float64
	Instances               float64
	DiskQuota               float64
	State                   string
	Command                 string
	HealthCheckType         string
	HealthCheckTimeout      float64
	HealthCheckHttpEndpoint string
	Diego                   bool
	EnableSsh               bool
	EnviornmentVar          map[string]interface{}
	ServiceNames            []interface{}
	URLs                    []interface{}
}

//Service representation
type Service struct {
	InstanceName string
	Label        string
	ServicePlan  string
	Type         string
	Credentials  map[string]interface{}
	SyslogDrain  string
}

type Quota struct {
	Name 					string
	NonBasicServicesAllowed	bool
	TotalServices			float64
	TotalRoutes				float64
	TotalPrivateDomain		float64
	MemoryLimit				float64
	TrialDBAllowed			bool
	InstanceMemoryLimit		float64
	AppInstanceLimit		float64
	AppTaskLimit			float64
	TotalServiceKeys		float64
	TotalReservedRoutePorts	float64
}

type SecurityGroup	struct {
	Name			string
	Rules			Rules
	RunningDefault	bool
	StagingDefault	bool
}

type Rule struct {
	Description		string
	Destination		string
	Log				bool
	Ports			string
	Protocol		string
}

type Orgs []Org
type Quotas map[string]Quota
type Rules	[]Rule
type SecurityGroups	[]SecurityGroup
type Spaces []Space
type Apps []App
type Services []Service

type ImportedOrg struct {
	Guid   string
	Name   string
	Spaces ISpaces
}

type ImportedSpace struct {
	Guid     string
	Name     string
	Apps     IApps
	Services IServices
}

type ImportedApp struct {
	Guid    string
	Name    string
	Droplet string
	Src     string
}

type ImportedService struct {
	Guid string
	Name string
}

type IServices []ImportedService
type IApps []ImportedApp
type ISpaces []ImportedSpace
type IOrgs []ImportedOrg

func (orgs *Orgs) ExportMetaOnly() string {
	writeToJson(*orgs)
	return "Succefully exported apps metadata to apps.json file."
}

func (orgs *Orgs) ExportMetaAndBits(apiHelper apihelper.CFAPIHelper) string {
	writeToJson(*orgs)
	//chBits := make(chan string, 2)
	var wg sync.WaitGroup
	i := 0
	for _, org := range *orgs {
		for _, space := range org.Spaces {
			i += len(space.Apps) * 2
			//download := (space.Name == "jigsheth")
			for _, app := range space.Apps {
				//if(download) {
				wg.Add(2)
				go apiHelper.GetBlob(org.Name,space.Name,"/v2/apps/"+app.Guid+"/droplet/download", url.PathEscape(app.Name)+"_"+app.Guid+".droplet", &wg)
				go apiHelper.GetBlob(org.Name,space.Name,"/v2/apps/"+app.Guid+"/download", url.PathEscape(app.Name)+"_"+app.Guid+".src", &wg)
				//}
			}
		}
	}
	log.Println("Number of app bits to download ", i)
	wg.Wait()
	//i = 4
	//for msg := range chBits {
	//	i -= 1
	//	log.Println("Wrote file: ", msg)
	//	if i == 0 {
	//		close(chBits)
	//	}
	//}
	return "Succefully exported apps metadata to apps.json file and downloaded all bits."
}

func ImportMetaAndBits(apiHelper apihelper.CFAPIHelper) string {
	orgs := readToJson()
	var iorgs IOrgs
	for _, org := range orgs {
		output, err := apiHelper.CheckOrg(org.Name, true)
		check(err)
		iorg := ImportedOrg{
			Guid: output.Guid,
			Name: output.Name,
		}
		var ispaces ISpaces
		for _, space := range org.Spaces {
			output, err := apiHelper.CheckSpace(space.Name, iorg.Guid, true)
			check(err)
			ispace := ImportedSpace{
				Guid: output.Guid,
				Name: output.Name,
			}
			var iservices IServices
			var rservices apihelper.IServices
			for _, service := range space.Services {
				mservice := apihelper.Service{
					InstanceName: service.InstanceName,
					Label:        service.Label,
					ServicePlan:  service.ServicePlan,
					Type:         service.Type,
					Credentials:  service.Credentials,
					SyslogDrain:  service.SyslogDrain,
				}
				output, err := apiHelper.CheckServiceInstance(mservice, ispace.Guid, true)
				check(err)
				iservice := ImportedService{
					Guid: output.Guid,
					Name: output.Name,
				}
				rservice := apihelper.ImportedService{
					Guid: output.Guid,
					Name: output.Name,
				}
				iservices = append(iservices, iservice)
				rservices = append(rservices, rservice)
			}
			ispace.Services = iservices
			var iapps IApps
			for _, app := range space.Apps {
				mapp := apihelper.App{
					Guid:                    app.Guid,
					Name:                    app.Name,
					Memory:                  app.Memory,
					Instances:               app.Instances,
					DiskQuota:               app.DiskQuota,
					State:                   app.State,
					Command:                 app.Command,
					HealthCheckType:         app.HealthCheckType,
					HealthCheckTimeout:      app.HealthCheckTimeout,
					HealthCheckHttpEndpoint: app.HealthCheckHttpEndpoint,
					Diego:          app.Diego,
					EnableSsh:      app.EnableSsh,
					EnviornmentVar: app.EnviornmentVar,
					URLs:           app.URLs,
					ServiceNames:   app.ServiceNames,
				}
				output, err := apiHelper.CheckApp(mapp, rservices, ispace.Guid, true)
				check(err)
				iapp := ImportedApp{
					Guid:    output.Guid,
					Name:    output.Name,
					Droplet: output.Droplet,
					Src:     output.Src,
				}
				iapps = append(iapps, iapp)
			}
			ispace.Apps = iapps
			ispaces = append(ispaces, ispace)
		}
		iorg.Spaces = ispaces
		iorgs = append(iorgs, iorg)
	}

	chBits := make(chan string, 2)
	i := 0
	for _, org := range iorgs {
		for _, space := range org.Spaces {
			i += len(space.Apps) * 2
			for _, app := range space.Apps {
				go apiHelper.PutBlob("/v2/apps/"+app.Guid+"/droplet/upload", app.Droplet, chBits)
				go apiHelper.PutBlob("/v2/apps/"+app.Guid+"/bits", app.Src, chBits)
			}
		}
	}
	fmt.Println("Number of app bits to upload ", i)
	for msg := range chBits {
		i -= 1
		fmt.Println(msg)
		if i == 0 {
			close(chBits)
		}
	}

	b, _ := json.MarshalIndent(iorgs, "", "\t")
	err := ioutil.WriteFile("imported_apps.json", b, 0644)
	check(err)
	return "Succefully imported apps metadata from apps.json file and uploaded all bits."
}

func writeToJson(orgs Orgs) {
	b, _ := json.MarshalIndent(orgs, "", "\t")
	err := ioutil.WriteFile("apps.json", b, 0644)
	check(err)
}
func readToJson() Orgs {
	var orgs Orgs
	b, err := ioutil.ReadFile("apps.json")
	check(err)
	err = json.Unmarshal(b, &orgs)
	check(err)
	return orgs
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
