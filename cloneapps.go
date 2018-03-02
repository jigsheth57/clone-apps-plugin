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
			Minor: 0,
			Build: 0,
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
		},
	}
}

//CloneAppsCmd doer
func (cmd *CloneAppsCmd) CloneAppsCmd(args []string) {
	flagVals := ParseFlags(args)

	var orgs []models.Org
	var err error
	var exportMeta models.Report

	if flagVals.OrgName != "" {
		org, err := cmd.getOrg(flagVals.OrgName)
		if nil != err {
			fmt.Println(err)
			os.Exit(1)
		}
		orgs = append(orgs, org)
	} else {
		orgs, err = cmd.getOrgs()
		if nil != err {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	exportMeta.Orgs = orgs
	if flagVals.Download == "download" {
		fmt.Println(exportMeta.MetaAndBits(cmd.apiHelper))
	} else {
		fmt.Println(exportMeta.MetaOnly())
	}
}

func (cmd *CloneAppsCmd) getOrgs() ([]models.Org, error) {
	rawOrgs, err := cmd.apiHelper.GetOrgs()
	if nil != err {
		return nil, err
	}

	var orgs = []models.Org{}

	for _, o := range rawOrgs {
		orgDetails, err := cmd.getOrgDetails(o)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, orgDetails)
	}
	return orgs, nil
}

func (cmd *CloneAppsCmd) getOrg(name string) (models.Org, error) {
	rawOrg, err := cmd.apiHelper.GetOrg(name)
	if nil != err {
		return models.Org{}, err
	}

	return cmd.getOrgDetails(rawOrg)
}

func (cmd *CloneAppsCmd) getOrgDetails(o apihelper.Organization) (models.Org, error) {
	quota, err := cmd.apiHelper.GetQuotaMemoryLimit(o.QuotaURL)
	if nil != err {
		return models.Org{}, err
	}
	spaces, err := cmd.getSpaces(o.SpacesURL)
	if nil != err {
		return models.Org{}, err
	}
	return models.Org{
		Name:        o.Name,
		MemoryQuota: int(quota),
		Spaces:      spaces,
	}, nil
}

func (cmd *CloneAppsCmd) getSpaces(spaceURL string) ([]models.Space, error) {
	rawSpaces, err := cmd.apiHelper.GetOrgSpaces(spaceURL)
	if nil != err {
		return nil, err
	}
	var spaces = []models.Space{}
	for _, s := range rawSpaces {
		apps, services, err := cmd.getAppsAndServices(s.SummaryURL)
		if nil != err {
			return nil, err
		}
		spaces = append(spaces,
			models.Space{
				Name: s.Name,
				Apps: apps,
				Services: services,
			},
		)
	}
	return spaces, nil
}

func (cmd *CloneAppsCmd) getAppsAndServices(summaryURL string) ([]models.App, []models.Service, error) {
	rawApps, rawServices, err := cmd.apiHelper.GetSpaceAppsAndServices(summaryURL)
	if nil != err {
		return nil, nil, err
	}
	var apps = []models.App{}
	var services = []models.Service{}
	for _, a := range rawApps {
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
			HealthCheckHttpEndpoint:a.HealthCheckHttpEndpoint,
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
	return apps, services, nil
}

//Run runs the plugin
func (cmd *CloneAppsCmd) Run(cli plugin.CliConnection, args []string) {
	if args[0] == "export-apps" {
		cmd.apiHelper = apihelper.New(cli)
		cmd.CloneAppsCmd(args)
	}
}

func main() {
	plugin.Start(new(CloneAppsCmd))
}
