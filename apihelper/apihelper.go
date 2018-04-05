package apihelper

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"io"
	"bytes"
	"os"
	"net/url"
	"net/http"
	"io/ioutil"
	"crypto/tls"
	"encoding/json"
	"mime/multipart"

	"github.com/cloudfoundry/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/jigsheth57/clone-apps-plugin/cfcurl"
)

var (
	ErrOrgNotFound = errors.New("organization not found")
)
var (
	ErrSharedDomainNotFound = errors.New("shared domain not found")
)
var (
	ErrManagedServiceNotFound = errors.New("managed service not found")
)
var (
	ErrManagedServicePlanNotFound = errors.New("managed service plan not found")
)

//Organization representation
type Organization struct {
	Name      string
	QuotaURL  string
	SpacesURL string
}

//Space representation
type Space struct {
	Name    	string
	SummaryURL	string
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

type Orgs []Organization
type Spaces []Space
type Apps []App
type Services []Service

type ImportedOrg struct {
	Guid	string
	Name	string
	Spaces	ISpaces
}

type ImportedSpace struct {
	Guid		string
	Name		string
	Apps		IApps
	Services	IServices
}

type ImportedApp struct {
	Guid	string
	Name	string
	Droplet	string
	Src		string
}

type ImportedService struct {
	Guid	string
	Name	string
}

type IServices []ImportedService
type IApps []ImportedApp
type ISpaces []ImportedSpace
type IOrgs []ImportedOrg


//CFAPIHelper to wrap cf curl results
type CFAPIHelper interface {
	GetOrgs() (Orgs, error)
	GetOrg(string) (Organization, error)
	GetDomainGuid(name string) (string, error)
	GetServiceInstanceGuid(name string, stype string) (string, error)
	GetQuotaMemoryLimit(string) (float64, error)
	GetOrgSpaces(string) (Spaces, error)
	GetSpaceAppsAndServices(string) (Apps, Services, error)
	GetBlob(blobURL string, filename string, c chan string)
	PutBlob(blobURL string, filename string, c chan string)
	CheckOrg(name string, create bool ) (ImportedOrg, error)
	CheckSpace(name string, orgguid string, create bool ) (ImportedSpace, error)
	CheckApp(mapp App, rservices IServices, spaceguid string, create bool ) (ImportedApp, error)
	CheckServiceInstance( service Service, spaceguid string, create bool ) (ImportedService, error)
}

//APIHelper implementation
type APIHelper struct {
	cli plugin.CliConnection
}

func New(cli plugin.CliConnection) CFAPIHelper {
	return &APIHelper{cli}
}

//GetOrgs returns a struct that represents critical fields in the JSON
func (api *APIHelper) GetOrgs() (Orgs, error) {
	orgsJSON, err := cfcurl.Curl(api.cli, "/v2/organizations")
	if nil != err {
		return nil, err
	}
	pages := int(orgsJSON["total_pages"].(float64))
	orgs := []Organization{}
	for i := 1; i <= pages; i++ {
		if 1 != i {
			orgsJSON, err = cfcurl.Curl(api.cli, "/v2/organizations?page="+strconv.Itoa(i))
		}
		for _, o := range orgsJSON["resources"].([]interface{}) {
			theOrg := o.(map[string]interface{})
			entity := theOrg["entity"].(map[string]interface{})
			name := entity["name"].(string)
			if (name == "system") {
				continue
			}
			orgs = append(orgs,
				Organization{
					Name:      name,
					QuotaURL:  entity["quota_definition_url"].(string),
					SpacesURL: entity["spaces_url"].(string),
				})
		}
	}
	return orgs, nil
}

//GetOrg returns a struct that represents critical fields in the JSON
func (api *APIHelper) GetOrg(name string) (Organization, error) {
	query := fmt.Sprintf("name:%s", name)
	path := fmt.Sprintf("/v2/organizations?q=%s", url.QueryEscape(query))
	orgsJSON, err := cfcurl.Curl(api.cli, path)
	if nil != err {
		return Organization{}, err
	}

	results := int(orgsJSON["total_results"].(float64))
	if results == 0 {
		return Organization{}, ErrOrgNotFound
	}

	orgResource := orgsJSON["resources"].([]interface{})[0]
	org := api.orgResourceToOrg(orgResource)

	return org, nil
}

func (api *APIHelper) orgResourceToOrg(o interface{}) Organization {
	theOrg := o.(map[string]interface{})
	entity := theOrg["entity"].(map[string]interface{})
	return Organization{
		Name:      entity["name"].(string),
		QuotaURL:  entity["quota_definition_url"].(string),
		SpacesURL: entity["spaces_url"].(string),
	}
}

//GetDomainGuid returns a shared domain guid
func (api *APIHelper) GetDomainGuid(name string) (string, error) {
	query := fmt.Sprintf("name:%s", name)
	path := fmt.Sprintf("/v2/shared_domains?q=%s", url.QueryEscape(query))
	domainJSON, err := cfcurl.Curl(api.cli, path)
	if nil != err {
		return "", err
	}

	results := int(domainJSON["total_results"].(float64))
	if results == 0 {
		return "", ErrSharedDomainNotFound
	}

	domainResource := domainJSON["resources"].([]interface{})[0]
	theDomain := domainResource.(map[string]interface{})
	metadata := theDomain["metadata"].(map[string]interface{})
	guid := metadata["guid"].(string)

	return guid, nil
}

//GetServiceInstanceGuid returns a service instance guid
func (api *APIHelper) GetServiceInstanceGuid(name string, stype string) (string, error) {
	query := fmt.Sprintf("name:%s", name)
	var path string
	if (stype == "managed") {
		path = fmt.Sprintf("/v2/service_instances?q=%s", url.QueryEscape(query))
	}
	if (stype == "user_provided") {
		path = fmt.Sprintf("/v2/user_provided_service_instances?q=%s", url.QueryEscape(query))
	}

	siJSON, err := cfcurl.Curl(api.cli, path)
	check(err)

	results := int(siJSON["total_results"].(float64))
	if results == 0 {
		return "", ErrManagedServiceNotFound
	}

	siResource := siJSON["resources"].([]interface{})[0]
	theSI := siResource.(map[string]interface{})
	metadata := theSI["metadata"].(map[string]interface{})
	guid := metadata["guid"].(string)

	return guid, nil
}

//getServicePlanGuid returns a managed service plan guid
func (api *APIHelper) getServicePlanGuid(label string, plan string) (string, error) {
	var guid string
	query := fmt.Sprintf("label:%s", label)
	path := fmt.Sprintf("/v2/services?q=%s", url.QueryEscape(query))
	serviceJSON, err := cfcurl.Curl(api.cli, path)
	check(err)
	results := int(serviceJSON["total_results"].(float64))
	if results == 0 {
		return "", ErrManagedServiceNotFound
	}
	resource := serviceJSON["resources"].([]interface{})[0]
	service := resource.(map[string]interface{})
	entity := service["entity"].(map[string]interface{})
	spurl := entity["service_plans_url"].(string)
	serviceplanJSON, err := cfcurl.Curl(api.cli, spurl)
	check(err)
	results = int(serviceplanJSON["total_results"].(float64))
	if results == 0 {
		return "", ErrManagedServicePlanNotFound
	}
	for _, sp := range serviceplanJSON["resources"].([]interface{}) {
		splan := sp.(map[string]interface{})
		entity := splan["entity"].(map[string]interface{})
		name := entity["name"].(string)
		if (name != plan) {
			continue
		}
		metadata := splan["metadata"].(map[string]interface{})
		guid = metadata["guid"].(string)
	}
	return guid, nil
}

//GetQuotaMemoryLimit retruns the amount of memory (in MB) that the org is allowed
func (api *APIHelper) GetQuotaMemoryLimit(quotaURL string) (float64, error) {
	quotaJSON, err := cfcurl.Curl(api.cli, quotaURL)
	if nil != err {
		return 0, err
	}
	return quotaJSON["entity"].(map[string]interface{})["memory_limit"].(float64), nil
}

//GetOrgSpaces returns the spaces in an org.
func (api *APIHelper) GetOrgSpaces(spacesURL string) (Spaces, error) {
	nextURL := spacesURL
	spaces := []Space{}
	for nextURL != "" {
		spacesJSON, err := cfcurl.Curl(api.cli, nextURL)
		if nil != err {
			return nil, err
		}
		for _, s := range spacesJSON["resources"].([]interface{}) {
			theSpace := s.(map[string]interface{})
			metadata := theSpace["metadata"].(map[string]interface{})
			entity := theSpace["entity"].(map[string]interface{})
			spaces = append(spaces,
				Space{
					Name:    entity["name"].(string),
					SummaryURL: metadata["url"].(string)+"/summary",
				})
		}
		if next, ok := spacesJSON["next_url"].(string); ok {
			nextURL = next
		} else {
			nextURL = ""
		}
	}
	return spaces, nil
}

//GetSpaceAppsAndServices returns the apps and the services in a space
func (api *APIHelper) GetSpaceAppsAndServices(summaryURL string) (Apps, Services, error) {
	apps := []App{}
	services := []Service{}
	summaryJSON, err := cfcurl.Curl(api.cli, summaryURL)
	if nil != err {
		return nil, nil, err
	}
	if _, ok := summaryJSON["apps"]; ok {
		for _, a := range summaryJSON["apps"].([]interface{}) {
			theApp := a.(map[string]interface{})
			httpEndpoint, err := theApp["health_check_http_endpoint"].(string)
			if err {
				httpEndpoint = ""
			}
			httpTimeout, err := theApp["health_check_timeout"].(float64)
			if err {
				httpTimeout = 180
			}
			environmentVar, ok := theApp["environment_json"].(map[string]interface{})
			if (ok) {
				if _, ok := environmentVar["redacted_message"]; ok {
					environmentVar = make(map[string]interface{})
				}
			}
			apps = append(apps,
				App{
					Guid: theApp["guid"].(string),
					Name: theApp["name"].(string),
					Memory:	theApp["memory"].(float64),
					Instances:	theApp["instances"].(float64),
					DiskQuota:	theApp["disk_quota"].(float64),
					State:	"STOPPED",
					Command:	theApp["detected_start_command"].(string),
					HealthCheckType:	theApp["health_check_type"].(string),
					HealthCheckTimeout:	httpTimeout,
					HealthCheckHttpEndpoint:	httpEndpoint,
					Diego: true,
					EnableSsh:	theApp["enable_ssh"].(bool),
					EnviornmentVar: environmentVar,
					ServiceNames: theApp["service_names"].([]interface{}),
					URLs: theApp["urls"].([]interface{}),
				})
		}
	}
	if _, ok := summaryJSON["services"]; ok {
		for _, s := range summaryJSON["services"].([]interface{}) {
			theService := s.(map[string]interface{})
			name := theService["name"].(string)
			if _, servicePlanExist := theService["service_plan"]; servicePlanExist {
				//if boundedApps := theService["bound_app_count"].(float64); boundedApps > 0 {
					servicePlan := theService["service_plan"].(map[string]interface{})
					if _, serviceExist := servicePlan["service"]; serviceExist {
						service := servicePlan["service"].(map[string]interface{})
						label := service["label"].(string)
						services = append(services,
							Service{
								InstanceName: name,
								Label:       label,
								ServicePlan: servicePlan["name"].(string),
								Type: "managed",
							})
					}
				//}
			} else {
				guid := theService["guid"].(string)
				instanceURL := "/v2/service_instances/"+guid
				cupsJSON, err := cfcurl.Curl(api.cli, instanceURL)
				if nil != err {
					return nil, nil, err
				}
				if _, ok := cupsJSON["entity"]; ok {
					entity := cupsJSON["entity"].(map[string]interface{})
					cred, ok := entity["credentials"].(map[string]interface{})
					if (ok) {
						if _, ok := cred["redacted_message"]; ok {
							cred = make(map[string]interface{})
						}
					}
					services = append(services,
						Service{
							InstanceName: name,
							Label:       "",
							ServicePlan: "",
							Type: "user_provided",
							Credentials: cred,
							SyslogDrain: entity["syslog_drain_url"].(string),
						})
				}
			}
		}
	}
	return apps, services, nil
}

//Download file
func (api *APIHelper) GetBlob(blobURL string, filename string, c chan string) {
	apiendpoint, err := api.cli.ApiEndpoint()
	if nil != err {
		return
	}
	//fmt.Println("URL: "+apiendpoint+blobURL)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", apiendpoint+blobURL, nil)
	accessToken, err := api.cli.AccessToken()
	if nil != err {
		return
	}
	req.Header.Set("Authorization",accessToken)
	res, _ := client.Do(req)
	//fmt.Println("HTTP_STATUS: "+res.Status)
	body, err := ioutil.ReadAll(res.Body)

	// write whole the body
	err = ioutil.WriteFile(filename, body, 0644)
	check(err)
	c <- filename
}

//Upload file
func (api *APIHelper) PutBlob(blobURL string, filename string, c chan string) {

	var msg string
	if(strings.Contains(blobURL,"droplet")) {
		msg, _ = putDroplet(api,blobURL,filename)
	}
	if(strings.Contains(blobURL,"bits")) {
		msg, _ = putSrc(api,blobURL,filename)
	}
	c <- msg
}

type orgInput struct {
	Name	string `json:"name"`
}
func (api *APIHelper) CheckOrg(name string, create bool ) (ImportedOrg, error) {
	var org plugin_models.GetOrg_Model
	var iorg ImportedOrg
	fmt.Println("Looking for org: "+name)
	org, err := api.cli.GetOrg(name)
	if nil != err && create {
		body := orgInput{
			Name:name,
		}
		bodyJSON, _ := json.Marshal(body)
		fmt.Println("Creating org: "+name+" with payload: "+string(bodyJSON))
		result, err := httpRequest(api,"POST","/v2/organizations",string(bodyJSON))
		if nil != err {
			fmt.Println("Error creating org: "+name)
			fmt.Println(err)
		}
		if nil != result {
			metadata := result["metadata"].(map[string]interface{})
			iorg = ImportedOrg{
				Name: name,
				Guid: metadata["guid"].(string),
			}
		}
	} else {
		fmt.Println("Found existing org: "+name)
		iorg = ImportedOrg{
			Name:name,
			Guid:org.Guid,
		}
	}
	return iorg, nil
}

type spaceInput struct {
	Name	string `json:"name"`
	Guid	string `json:"organization_guid"`
}
func (api *APIHelper) CheckSpace(name string, orgguid string, create bool ) (ImportedSpace, error) {
	var ispace ImportedSpace
	fmt.Println("Looking for space: "+name)
	query := fmt.Sprintf("name:%s", name)
	path := fmt.Sprintf("/v2/organizations/"+orgguid+"/spaces?q=%s", url.QueryEscape(query))
	spaceJSON, err := cfcurl.Curl(api.cli, path)
	if nil != err && create {
		body := spaceInput{
			Name:name,
			Guid:orgguid,
		}
		bodyJSON, _ := json.Marshal(body)
		fmt.Println("Creating space ("+name+") with payload: "+string(bodyJSON))
		result, err := httpRequest(api,"POST","/v2/spaces",string(bodyJSON))
		if nil != err {
			fmt.Println("Error creating space: "+name)
			fmt.Println(err)
			ispace = ImportedSpace{
				Name: name,
			}
		}
		if nil != result {
			metadata := result["metadata"].(map[string]interface{})
			ispace = ImportedSpace{
				Name: name,
				Guid: metadata["guid"].(string),
			}
		}
	} else {
		results := int(spaceJSON["total_results"].(float64))
		if results != 0 {
			spaceResource := spaceJSON["resources"].([]interface{})[0]
			theSpace := spaceResource.(map[string]interface{})
			metadata := theSpace["metadata"].(map[string]interface{})
			fmt.Println("Found existing space: "+name)
			ispace = ImportedSpace{
				Name:name,
				Guid:metadata["guid"].(string),
			}
		}
	}
	return ispace, nil
}

type serviceInput struct {
	Name		string `json:"name"`
	SpaceGuid	string `json:"space_guid"`
	ServicePlanGuid	string `json:"service_plan_guid"`
}
type cupsInput struct {
	Name		string `json:"name"`
	SpaceGuid	string `json:"space_guid"`
	Credentials		map[string]interface{} `json:"credentials"`
	SyslogDrain		string `json:"syslog_drain_url"`
}
func (api *APIHelper) CheckServiceInstance( service Service, spaceguid string, create bool ) (ImportedService, error) {
	var iservice ImportedService
	if service.Type == "managed" {
		spguid, err := api.getServicePlanGuid(service.Label,service.ServicePlan)
		if len(spguid) < 1 {
			return iservice, ErrManagedServicePlanNotFound
		}
		check(err)
		siguid, err := api.GetServiceInstanceGuid(service.InstanceName,service.Type)
		if len(siguid) > 1 {
			create = false
			iservice = ImportedService{
				Name: service.InstanceName,
				Guid: siguid,
			}
			fmt.Println("Service instance "+service.InstanceName+" found.")
		}
		if create {
			body := serviceInput{
				Name:service.InstanceName,
				SpaceGuid:spaceguid,
				ServicePlanGuid:spguid,
			}
			bodyJSON, _ := json.Marshal(body)
			fmt.Println("Creating service instance "+service.InstanceName+" with payload: "+string(bodyJSON))
			result, err := httpRequest(api,"POST","/v2/service_instances?accepts_incomplete=true",string(bodyJSON))
			if nil != err {
				fmt.Println("Error creating service instance: "+service.InstanceName)
				fmt.Println(err)
			}
			if nil != result {
				metadata := result["metadata"].(map[string]interface{})
				iservice = ImportedService{
					Name: service.InstanceName,
					Guid: metadata["guid"].(string),
				}
				fmt.Println("Service instance "+service.InstanceName+" created.")
			}
		}
	}
	if service.Type == "user_provided" {
		siguid, _ := api.GetServiceInstanceGuid(service.InstanceName,service.Type)
		if len(siguid) > 1 {
			create = false
			iservice = ImportedService{
				Name: service.InstanceName,
				Guid: siguid,
			}
			fmt.Println("Service instance "+service.InstanceName+" found.")
		}
		if create {
			body := cupsInput{
				Name:service.InstanceName,
				SpaceGuid:spaceguid,
				Credentials:service.Credentials,
				SyslogDrain:service.SyslogDrain,
			}
			bodyJSON, _ := json.Marshal(body)
			fmt.Println("Creating service instance "+service.InstanceName+" with payload: "+string(bodyJSON))
			result, err := httpRequest(api,"POST","/v2/user_provided_service_instances",string(bodyJSON))
			if nil != err {
				fmt.Println("Error creating service instance: "+service.InstanceName)
				fmt.Println(err)
			}
			if nil != result {
				metadata := result["metadata"].(map[string]interface{})
				iservice = ImportedService{
					Name: service.InstanceName,
					Guid: metadata["guid"].(string),
				}
				fmt.Println("Service instance "+service.InstanceName+" created.")

			}
		}
	}

	return iservice, nil
}

type serviceBindingInput struct {
	ServiceInstanceGuid	string `json:"service_instance_guid"`
	AppGuid				string `json:"app_guid"`
}
func (api *APIHelper) bindService(siguid string, appguid string ) (error) {
	body := serviceBindingInput{
		ServiceInstanceGuid:siguid,
		AppGuid:appguid,
	}
	bodyJSON, _ := json.Marshal(body)
	_, err := httpRequest(api,"POST","/v2/service_bindings",string(bodyJSON))
	if nil != err {
		fmt.Println("Problem binding service instance ("+siguid+") to app instance ("+appguid+"): ")
		fmt.Println(err)
	}
	return nil
}

type appInput struct {
	SpaceGuid				string `json:"space_guid"`
	Name					string `json:"name"`
	Memory					float64 `json:"memory"`
	Instances				float64 `json:"instances"`
	DiskQuota   			float64 `json:"disk_quota"`
	State					string `json:"state"`
	Command					string `json:"command"`
	HealthCheckType			string `json:"health_check_type"`
	HealthCheckTimeout		float64 `json:"health_check_timeout"`
	HealthCheckHttpEndpoint	string `json:"health_check_http_endpoint"`
	Diego					bool `json:"diego"`
	EnableSsh				bool `json:"enable_ssh"`
	EnviornmentVar			map[string]interface{} `json:"environment_json"`
}
func (api *APIHelper) CheckApp(mapp App, rservices IServices, spaceguid string, create bool ) (ImportedApp, error) {
	var app plugin_models.GetAppModel
	var iapp ImportedApp
	fmt.Println("Looking for app: "+mapp.Name)
	app, err := api.cli.GetApp(mapp.Name)
	if nil != err && create {
		body := appInput{
			SpaceGuid:spaceguid,
			Name:mapp.Name,
			Memory:mapp.Memory,
			Instances:mapp.Instances,
			DiskQuota:mapp.DiskQuota,
			State:mapp.State,
			Command:mapp.Command,
			HealthCheckType:mapp.HealthCheckType,
			HealthCheckTimeout:180,
			HealthCheckHttpEndpoint:mapp.HealthCheckHttpEndpoint,
			Diego:mapp.Diego,
			EnableSsh:mapp.EnableSsh,
			EnviornmentVar:mapp.EnviornmentVar,
		}
		bodyJSON, _ := json.Marshal(body)
		fmt.Println("Creating app ("+mapp.Name+") with payload: "+string(bodyJSON))
		result, err := httpRequest(api,"POST","/v2/apps",string(bodyJSON))
		if nil != err {
			fmt.Println("Error creating app: "+mapp.Name)
			fmt.Println(err)
		}
		if nil != result {
			metadata := result["metadata"].(map[string]interface{})
			iapp = ImportedApp{
				Name:mapp.Name,
				Guid:metadata["guid"].(string),
				Droplet: url.PathEscape(mapp.Name)+"_"+mapp.Guid+".droplet",
				Src: url.PathEscape(mapp.Name)+"_"+mapp.Guid+".src",
			}
			fmt.Println("App "+mapp.Name+" created.")
		}
		for _, url := range mapp.URLs {
			s := strings.SplitN(url.(string),".",2)
			domainguid, err := api.GetDomainGuid(s[1])
			check(err)
			routeguid, err := api.createRoute(domainguid,spaceguid,s[0])
			check(err)
			fmt.Println("Route ("+url.(string)+") created.")
			api.bindRoute(routeguid,iapp.Guid)
			fmt.Println("Route ("+url.(string)+") bounded to app "+mapp.Name+".")
		}
		for _, siname := range mapp.ServiceNames {
			siguid, err := getServiceInstanceGuid(rservices, siname.(string))
			check(err)
			api.bindService(siguid,iapp.Guid)
			fmt.Println("Service instance ("+siname.(string)+") bounded to app "+mapp.Name+".")
		}
	} else {
		fmt.Println("Found existing app: "+mapp.Name)
		iapp = ImportedApp{
			Name:mapp.Name,
			Guid:app.Guid,
			Droplet: url.PathEscape(mapp.Name)+"_"+mapp.Guid+".droplet",
			Src: url.PathEscape(mapp.Name)+"_"+mapp.Guid+".src",
		}
	}
	return iapp, nil
}

func getServiceInstanceGuid (rservices IServices, name string) (string, error) {
	for _, service := range rservices {
		if (service.Name == name) {
			return service.Guid, nil
		}
	}
	return "", ErrManagedServiceNotFound
}

type routeInput struct {
	DomainGuid	string `json:"domain_guid"`
	SpaceGuid	string `json:"space_guid"`
	Hostname	string `json:"host"`
}
func (api *APIHelper) createRoute(domainguid string, spaceguid string, hostname string ) (string, error) {
	var rguid string
	create := false
	query1 := fmt.Sprintf("host:%s", hostname)
	query2 := fmt.Sprintf("domain_guid:%s", domainguid)
	path := fmt.Sprintf("/v2/routes?q=%s;q=%s", url.QueryEscape(query1), url.QueryEscape(query2))
	routeJSON, err := cfcurl.Curl(api.cli, path)
	if nil != err {
		create = true
	} else {
		results := int(routeJSON["total_results"].(float64))
		if results != 0 {
			routeResource := routeJSON["resources"].([]interface{})[0]
			theRoute := routeResource.(map[string]interface{})
			metadata := theRoute["metadata"].(map[string]interface{})
			rguid = metadata["guid"].(string)
			create = false
			fmt.Println("Found existing route with hostname: "+hostname)
		}
	}
	if create {
		body := routeInput{
			DomainGuid:domainguid,
			SpaceGuid:spaceguid,
			Hostname:hostname,
		}
		bodyJSON, _ := json.Marshal(body)
		fmt.Println("Creating route with payload: "+string(bodyJSON))
		result, err := httpRequest(api,"POST","/v2/routes",string(bodyJSON))
		if nil != err {
			fmt.Println("Error creating route: "+hostname)
			fmt.Println(err)
		}
		if nil != result {
			metadata := result["metadata"].(map[string]interface{})
			rguid = metadata["guid"].(string)
		}
	}
	return rguid, nil
}

func (api *APIHelper) bindRoute(routeguid string, appguid string ) (error) {
	httpRequest(api,"PUT","/v2/routes/"+routeguid+"/apps/"+appguid,"")
	return nil
}

func httpRequest(api *APIHelper, method string, url string, body string) (map[string]interface{}, error) {
	apiendpoint, err := api.cli.ApiEndpoint()
	check(err)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest(method, apiendpoint+url, strings.NewReader(body))
	accessToken, err := api.cli.AccessToken()
	check(err)
	req.Header.Set("Authorization",accessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	check(err)
	stscode := res.StatusCode
	//fmt.Println(stscode)
	response, err := ioutil.ReadAll(res.Body)
	if (stscode >= 400) {
		return nil, errors.New(string(response))
	}
	var f interface{}
	err = json.Unmarshal(response, &f)
	check(err)

	return f.(map[string]interface{}), nil
}

func putDroplet(api *APIHelper, url string, filename string) (string, error) {
	if _, err := os.Stat(filename); err == nil {
		apiendpoint, err := api.cli.ApiEndpoint()
		check(err)
		client := &http.Client{}
		accessToken, err := api.cli.AccessToken()
		check(err)

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)

		fileWriter, err := bodyWriter.CreateFormFile("droplet", filename)
		check(err)
		// open file handle
		fh, err := os.Open(filename)
		check(err)
		defer fh.Close()

		//iocopy
		_, err = io.Copy(fileWriter, fh)
		check(err)
		contentType := bodyWriter.FormDataContentType()
		bodyWriter.Close()

		req, _ := http.NewRequest("PUT", apiendpoint+url, bodyBuf)
		req.Header.Set("Authorization",accessToken)
		req.Header.Set("Content-Type", contentType)
		resp, err := client.Do(req)
		check(err)
		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		check(err)
		return "Uploaded droplet "+filename+" ("+resp.Status+")", nil
	} else {
		return "Expected droplet "+filename+" doesn't exist in working directory!", nil
	}
}

func putSrc(api *APIHelper, url string, filename string) (string, error) {
	if _, err := os.Stat(filename); err == nil {
		apiendpoint, err := api.cli.ApiEndpoint()
		check(err)
		client := &http.Client{}
		accessToken, err := api.cli.AccessToken()
		check(err)

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)

		fileWriter, err := bodyWriter.CreateFormFile("application", filename)
		check(err)
		// open file handle
		fh, err := os.Open(filename)
		check(err)
		defer fh.Close()

		//iocopy
		_, err = io.Copy(fileWriter, fh)
		check(err)
		contentType := bodyWriter.FormDataContentType()

		bodyWriter.WriteField("resources","[]")
		bodyWriter.Close()

		req, _ := http.NewRequest("PUT", apiendpoint+url, bodyBuf)
		req.Header.Set("Authorization",accessToken)
		req.Header.Set("Content-Type", contentType)
		resp, err := client.Do(req)
		check(err)
		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		check(err)
		return "Uploaded src "+filename+" ("+resp.Status+")", nil
	} else {
		return "Expected src "+filename+" doesn't exist in working directory!", nil
	}

}

func check(e error) {
	if e != nil {
		fmt.Println(e)
		panic(e)
	}
}