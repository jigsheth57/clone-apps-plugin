package models

import (
	"encoding/json"
	"io/ioutil"
	"github.com/jigsheth57/clone-apps-plugin/apihelper"
	"fmt"
)

type Org struct {
	Name        string
	MemoryQuota int
	Spaces      Spaces
}

type Space struct {
	Name string
	Apps Apps
	Services Services
}

//App representation
type App struct {
	Guid					string
	Name					string
	Memory					float64
	Instances				float64
	DiskQuota   			float64
	State					string
	Command					string
	HealthCheckType			string
	HealthCheckTimeout		float64
	HealthCheckHttpEndpoint	string
	Diego					bool
	EnableSsh				bool
	EnviornmentVar			map[string]interface{}
	ServiceNames			[]interface{}
	URLs					[]interface{}
}

//Service representation
type Service struct {
	InstanceName	string
	Label    		string
	ServicePlan 	string
	Type			string
	Credentials		map[string]interface{}
	SyslogDrain		string
}

type Orgs []Org
type Spaces []Space
type Apps []App
type Services []Service

type Report struct {
	Orgs Orgs
}

func (report *Report) MetaOnly() string {
	writeToJson(report.Orgs)
	return "Succefully exported apps metadata to apps.json file."
}

func (report *Report) MetaAndBits(apiHelper apihelper.CFAPIHelper) string {
	writeToJson(report.Orgs)
	chBits := make(chan string, 10)
	i := 0
	for _, org := range report.Orgs {
		for _, space := range org.Spaces {
			i += len(space.Apps)*2
			//download := (space.Name == "jigsheth")
			for _, app := range space.Apps {
				//if(download) {
					go apiHelper.GetBlob("/v2/apps/"+app.Guid+"/droplet/download", app.Name+"_"+app.Guid+".droplet", chBits)
					go apiHelper.GetBlob("/v2/apps/"+app.Guid+"/download", app.Name+"_"+app.Guid+".src", chBits)
				//}
			}
		}
	}
	fmt.Println("Number of app bits to download ", i)
	//i = 4
	for msg := range chBits {
		i -= 1
		fmt.Println("Wrote file: ", msg)
		if(i==0) {
			close(chBits)
		}
	}
	return "Succefully exported apps metadata to apps.json file and downloaded all bits."
}

func writeToJson(orgs Orgs) {
	b, _ := json.Marshal(orgs)
	err := ioutil.WriteFile("apps.json", b, 0644)
	check(err)
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}