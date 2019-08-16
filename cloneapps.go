package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/jigsheth57/clone-apps-plugin/apihelper"
	"github.com/jigsheth57/clone-apps-plugin/models"
)

//CloneAppsCmd the plugin
type CloneAppsCmd struct {
	apiHelper apihelper.CFAPIHelper
}

// contains CLI flag values
type flagVal struct {
	OrgName string
	Download string
}

func ParseFlags(args []string) flagVal {
	flagSet := flag.NewFlagSet(args[0], flag.ContinueOnError)

	// Create flags
	orgName := flagSet.String("o", "", "-o orgName")
	bits := flagSet.String("d", "", "-d download")

	err := flagSet.Parse(args[1:])
	if err != nil {

	}

	return flagVal{
		OrgName: string(*orgName),
		Download:  string(*bits),
	}
}

//GetMetadata returns metatada
func (cmd *CloneAppsCmd) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "clone-apps",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 2,
			Build: 28,
		},
		Commands: []plugin.Command{
			{
				Name:     "export-apps",
				HelpText: "Export apps metadata (including service instances info), droplets & src code",
				UsageDetails: plugin.Usage{
					Usage: "cf export-apps [-o orgName] [-d download]",
					Options: map[string]string{
						"o": "organization",
						"d": "download",
					},
				},
			},
			{
				Name:     "import-apps",
				HelpText: "Import apps metadata (including service instances info), droplets & src code",
				UsageDetails: plugin.Usage{
					Usage: "cf import-apps [-o orgName]",
					Options: map[string]string{
						"o": "organization",
					},
				},
			},
		},
	}
}

//ExportAppsCmd doer
func (cmd *CloneAppsCmd) ExportAppsCmd(args []string) {
	flagVals := ParseFlags(args)

	var orgs models.Orgs
	var quotas models.Quotas
	var err error

	quotas, err = cmd.getOrgQuota()
	if flagVals.OrgName != "" {
		org, err := cmd.getOrg(flagVals.OrgName, quotas)
		if nil != err {
			fmt.Println(err)
			os.Exit(1)
		}
		orgs = append(orgs, org)
	} else {
		orgs, err = cmd.getOrgs(quotas)
		if nil != err {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if flagVals.Download == "download" {
		fmt.Println(orgs.ExportMetaAndBits(cmd.apiHelper))
	} else {
		fmt.Println(orgs.ExportMetaOnly())
	}
}

func (cmd *CloneAppsCmd) ImportAppsCmd(args []string) {
	fmt.Println(models.ImportMetaAndBits(cmd.apiHelper))
}

func (cmd *CloneAppsCmd) getOrgQuota() (models.Quotas, error) {
	rawQuotas, err := cmd.apiHelper.GetOrgQuota()
	if nil != err {
		return nil, err
	}

	var quotas = models.Quotas{}

	for key, q := range rawQuotas {
		quotas[key] =
			models.Quota{
				Name:       				q.Name,
				NonBasicServicesAllowed:	q.NonBasicServicesAllowed,
				TotalServices:				q.TotalServices,
				TotalRoutes:				q.TotalRoutes,
				TotalPrivateDomain:			q.TotalPrivateDomain,
				MemoryLimit:				q.MemoryLimit,
				TrialDBAllowed:				q.TrialDBAllowed,
				InstanceMemoryLimit:		q.InstanceMemoryLimit,
				AppInstanceLimit:			q.AppInstanceLimit,
				AppTaskLimit:				q.AppTaskLimit,
				TotalServiceKeys:			q.TotalServiceKeys,
				TotalReservedRoutePorts:	q.TotalReservedRoutePorts,
			}
	}
	return quotas, nil
}

func (cmd *CloneAppsCmd) getOrgs(quotas models.Quotas) ([]models.Org, error) {
	rawOrgs, err := cmd.apiHelper.GetOrgs()
	if nil != err {
		return nil, err
	}

	var orgs = []models.Org{}

	for _, o := range rawOrgs {
		orgDetails, err := cmd.getOrgDetails(o,quotas)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, orgDetails)
	}
	return orgs, nil
}

func (cmd *CloneAppsCmd) getOrg(name string, quotas models.Quotas) (models.Org, error) {
	rawOrg, err := cmd.apiHelper.GetOrg(name)
	if nil != err {
		return models.Org{}, err
	}

	return cmd.getOrgDetails(rawOrg, quotas)
}

func (cmd *CloneAppsCmd) getOrgDetails(o apihelper.Organization, quotas models.Quotas) (models.Org, error) {
	var quota = models.Quota{}
	if q, found := quotas[o.QuotaGUID]; found {
		quota = q
	}
	spaces, err := cmd.getSpaces(o.SpacesURL)
	if nil != err {
		return models.Org{}, err
	}
	return models.Org{
		Name:       o.Name,
		Quota: 		quota,
		Spaces:     spaces,
	}, nil
}

func (cmd *CloneAppsCmd) getSpaces(spaceURL string) ([]models.Space, error) {
	rawSpaces, err := cmd.apiHelper.GetOrgSpaces(spaceURL)
	if nil != err {
		return nil, err
	}
	var spaces = []models.Space{}
	for _, s := range rawSpaces {
		apps, services, securityGroups, stagingSecurityGroups, err := cmd.getAppsAndServices(s)
		if nil != err {
			return nil, err
		}
		spaces = append(spaces,
			models.Space{
				Name: s.Name,
				Apps: apps,
				Services: services,
				SecurityGroup: securityGroups,
				StagingSecurityGroup: stagingSecurityGroups,
			},
		)
	}
	return spaces, nil
}

func (cmd *CloneAppsCmd) getAppsAndServices(space apihelper.Space) ([]models.App, []models.Service, []models.SecurityGroup, []models.SecurityGroup, error) {
	rawApps, rawServices, rawSecurityGroups, rawStagingSecurityGroups, err := cmd.apiHelper.GetSpaceAppsAndServices(space)
	if nil != err {
		return nil, nil, nil, nil, err
	}
	var apps = []models.App{}
	var services = []models.Service{}
	var securityGroups = []models.SecurityGroup{}
	var stagingSecurityGroups = []models.SecurityGroup{}
	for _, a := range rawApps {
		endpoint := a.HealthCheckHttpEndpoint
		if (a.HealthCheckType == "http" && endpoint == "") {
			endpoint = "/"
		}
		apps = append(apps, models.App{
			Guid: a.Guid,
			Name: a.Name,
			Memory:a.Memory,
			Instances:a.Instances,
			DiskQuota:a.DiskQuota,
			State:a.State,
			Command:a.Command,
			HealthCheckType:a.HealthCheckType,
			HealthCheckTimeout:a.HealthCheckTimeout,
			HealthCheckHttpEndpoint:endpoint,
			Diego:a.Diego,
			EnableSsh:a.EnableSsh,
			EnviornmentVar:a.EnviornmentVar,
			ServiceNames:a.ServiceNames,
			URLs:a.URLs,
		})
	}
	for _, s := range rawServices {
		services = append(services, models.Service{
			InstanceName: s.InstanceName,
			Label: s.Label,
			ServicePlan: s.ServicePlan,
			Type:s.Type,
			Credentials:s.Credentials,
			SyslogDrain:s.SyslogDrain,
		})
	}
	for _, sg := range rawSecurityGroups {
		rules := []models.Rule{}
		for _, r := range sg.Rules {
			rules = append(rules,
				models.Rule{
					Description: r.Description,
					Destination: r.Destination,
					Log:         r.Log,
					Ports:       r.Ports,
					Protocol:    r.Protocol,
				})
		}
		securityGroups = append(securityGroups, models.SecurityGroup{
			Name:           sg.Name,
			Rules:          rules,
			RunningDefault: sg.RunningDefault,
			StagingDefault: sg.StagingDefault,
		})
	}
	for _, ssg := range rawStagingSecurityGroups {
		rules := []models.Rule{}
		for _, r := range ssg.Rules {
			rules = append(rules,
				models.Rule{
					Description: r.Description,
					Destination: r.Destination,
					Log:         r.Log,
					Ports:       r.Ports,
					Protocol:    r.Protocol,
				})
		}
		stagingSecurityGroups = append(stagingSecurityGroups, models.SecurityGroup{
			Name:           ssg.Name,
			Rules:          rules,
			RunningDefault: ssg.RunningDefault,
			StagingDefault: ssg.StagingDefault,
		})
	}

	return apps, services, securityGroups, stagingSecurityGroups, nil
}

//Run runs the plugin
func (cmd *CloneAppsCmd) Run(cli plugin.CliConnection, args []string) {
	if args[0] == "export-apps" {
		cmd.apiHelper = apihelper.New(cli)
		cmd.ExportAppsCmd(args)
	}
	if args[0] == "import-apps" {
		cmd.apiHelper = apihelper.New(cli)
		cmd.ImportAppsCmd(args)
	}
}

func main() {
	plugin.Start(new(CloneAppsCmd))
}
