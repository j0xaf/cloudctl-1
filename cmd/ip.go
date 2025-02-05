package cmd

import (
	"fmt"

	"github.com/fi-ts/cloud-go/api/client/ip"

	"github.com/fi-ts/cloud-go/api/models"
	"github.com/fi-ts/cloudctl/cmd/helper"
	"github.com/fi-ts/cloudctl/cmd/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newIPCmd(c *config) *cobra.Command {
	ipCmd := &cobra.Command{
		Use:   "ip",
		Short: "manage ips",
		Long:  "TODO",
	}
	ipListCmd := &cobra.Command{
		Use:     "list",
		Short:   "list ips",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ipList()
		},
		PreRun: bindPFlags,
	}
	ipStaticCmd := &cobra.Command{
		Use:   "static <ip>",
		Short: "make an ephemeral ip static such that it won't be deleted if not used anymore",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ipStatic(args)
		},
		PreRun: bindPFlags,
	}
	ipAllocateCmd := &cobra.Command{
		Use:   "allocate <ip>",
		Short: "allocate a static IP address for your project that can be used for your cluster's service type load balancer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ipAllocate()
		},
		PreRun: bindPFlags,
	}
	ipFreeCmd := &cobra.Command{
		Use:     "free <ip>",
		Aliases: []string{"rm", "destroy", "remove", "delete"},
		Short:   "free an ip",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ipFree(args)
		},
		PreRun: bindPFlags,
	}

	ipCmd.AddCommand(ipListCmd)
	ipCmd.AddCommand(ipStaticCmd)
	ipCmd.AddCommand(ipFreeCmd)
	ipCmd.AddCommand(ipAllocateCmd)

	ipListCmd.Flags().StringP("ipaddress", "", "", "ipaddress to filter [optional]")
	ipListCmd.Flags().StringP("project", "", "", "project to filter [optional]")
	ipListCmd.Flags().StringP("prefix", "", "", "prefix to filter [optional]")
	ipListCmd.Flags().StringP("machineid", "", "", "machineid to filter [optional]")
	ipListCmd.Flags().StringP("network", "", "", "network to filter [optional]")

	must(ipListCmd.RegisterFlagCompletionFunc("project", c.comp.ProjectListCompletion))
	must(ipListCmd.RegisterFlagCompletionFunc("network", c.comp.NetworkListCompletion))

	ipStaticCmd.Flags().StringP("name", "", "", "set name of the ip address [required]")
	ipStaticCmd.Flags().StringP("description", "", "", "set description of the ip address [required]")
	must(ipStaticCmd.MarkFlagRequired("name"))
	must(ipStaticCmd.MarkFlagRequired("description"))

	ipAllocateCmd.Flags().StringP("name", "", "", "set name of the ip address [required]")
	ipAllocateCmd.Flags().StringP("description", "", "", "set description of the ip address [required]")
	ipAllocateCmd.Flags().StringP("specific-ip", "", "", "try allocating a specific ip address from a network [optional]")
	ipAllocateCmd.Flags().StringP("network", "", "", "the network of the ip address [required]")
	ipAllocateCmd.Flags().StringP("project", "", "", "the project of the ip address [required]")
	ipAllocateCmd.Flags().StringSliceP("tags", "", []string{}, "set tags of the ip address [optional]")
	must(ipAllocateCmd.MarkFlagRequired("name"))
	must(ipAllocateCmd.MarkFlagRequired("description"))
	must(ipAllocateCmd.MarkFlagRequired("network"))
	must(ipAllocateCmd.MarkFlagRequired("project"))
	must(ipAllocateCmd.RegisterFlagCompletionFunc("project", c.comp.ProjectListCompletion))

	return ipCmd
}

func (c *config) ipList() error {
	if helper.AtLeastOneViperStringFlagGiven("ipaddress", "project", "prefix", "machineid", "network") {
		params := ip.NewFindIPsParams()
		ifr := &models.V1IPFindRequest{
			IPAddress:        helper.ViperString("ipaddress"),
			ProjectID:        helper.ViperString("project"),
			ParentPrefixCidr: helper.ViperString("prefix"),
			NetworkID:        helper.ViperString("network"),
			MachineID:        helper.ViperString("machineid"),
		}
		params.SetBody(ifr)
		resp, err := c.cloud.IP.FindIPs(params, nil)
		if err != nil {
			return err
		}
		return output.New().Print(resp.Payload)
	}
	resp, err := c.cloud.IP.ListIPs(nil, nil)
	if err != nil {
		return err
	}
	return output.New().Print(resp.Payload)
}

func (c *config) ipStatic(args []string) error {
	ipAddress, err := c.getIPFromArgs(args)
	if err != nil {
		return err
	}

	params := ip.NewUpdateIPParams()
	iur := &models.V1IPUpdateRequest{
		Ipaddress: &ipAddress,
		Type:      "static",
	}
	if helper.ViperString("name") != nil {
		iur.Name = *helper.ViperString("name")
	}
	if helper.ViperString("description") != nil {
		iur.Description = *helper.ViperString("description")
	}

	if !viper.GetBool("yes-i-really-mean-it") {
		fmt.Println("Turning an IP from ephemeral to static is irreversible. The IP address is not cleaned up automatically on cluster deletion. The address will be accounted until the IP address gets freed manually from your side.")
		err = helper.Prompt("Are you sure? (y/n)", "y")
		if err != nil {
			return err
		}
	}

	params.SetBody(iur)
	resp, err := c.cloud.IP.UpdateIP(params, nil)
	if err != nil {
		return err
	}
	return output.New().Print(resp.Payload)
}

func (c *config) ipAllocate() error {
	params := ip.NewAllocateIPParams()
	iar := &models.V1IPAllocateRequest{
		Name:        *helper.ViperString("name"),
		Description: *helper.ViperString("description"),
		Type:        "static",
		Networkid:   helper.ViperString("network"),
		Projectid:   helper.ViperString("project"),
		Tags:        helper.ViperStringSlice("tags"),
	}

	if helper.ViperString("specific-ip") != nil {
		iar.Ipaddress = helper.ViperString("specific-ip")
	}

	if !viper.GetBool("yes-i-really-mean-it") {
		fmt.Println("Allocating a static IP address costs additional money because addresses are limited. The IP address is not cleaned up automatically on cluster deletion. The address will be accounted until the IP address gets freed manually from your side.")
		err := helper.Prompt("Are you sure? (y/n)", "y")
		if err != nil {
			return err
		}
	}

	params.SetBody(iar)
	resp, err := c.cloud.IP.AllocateIP(params, nil)
	if err != nil {
		return err
	}
	return output.New().Print(resp.Payload)
}

func (c *config) ipFree(args []string) error {
	ipAddress, err := c.getIPFromArgs(args)
	if err != nil {
		return err
	}

	params := ip.NewFreeIPParams()
	params.SetIP(ipAddress)
	resp, err := c.cloud.IP.FreeIP(params, nil)
	if err != nil {
		return err
	}

	return output.New().Print(resp.Payload)
}

func (c *config) getIPFromArgs(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("no ip given")
	}

	ipAddress := args[0]
	params := ip.NewGetIPParams()
	params.SetIP(ipAddress)

	_, err := c.cloud.IP.GetIP(params, nil)
	if err != nil {
		return "", err
	}
	return ipAddress, nil
}
