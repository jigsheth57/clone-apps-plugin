package apihelper

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/jigsheth57/clone-apps-plugin/cfcurl"
	"net/http"
	"io/ioutil"
)

var (
	ErrOrgNotFound = errors.New("organization not found")
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


//CFAPIHelper to wrap cf curl results
type CFAPIHelper interface {
	GetOrgs() (Orgs, error)
	GetOrg(string) (Organization, error)
	GetQuotaMemoryLimit(string) (float64, error)
	GetOrgSpaces(string) (Spaces, error)
	GetSpaceAppsAndServices(string) (Apps, Services, error)
	GetBlob(blobURL string, filename string, c chan string)
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
				if boundedApps := theService["bound_app_count"].(float64); boundedApps > 0 {
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
				}
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
	client := &http.Client{}
	req, _ := http.NewRequest("GET", apiendpoint+blobURL, nil)
	accessToken, err := api.cli.AccessToken()
	if nil != err {
		return
	}
	req.Header.Set("Authorization",accessToken)
	res, _ := client.Do(req)
	body, err := ioutil.ReadAll(res.Body)

	// write whole the body
	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		panic(err)
	}
	c <- filename
}