package cmd

import (
	"fmt"

	"github.com/fi-ts/cloud-go/api/client/volume"

	"github.com/fi-ts/cloud-go/api/models"
	"github.com/fi-ts/cloudctl/cmd/helper"
	"github.com/fi-ts/cloudctl/cmd/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newVolumeCmd(c *config) *cobra.Command {
	volumeCmd := &cobra.Command{
		Use:   "volume",
		Short: "manage volume",
		Long:  "list/find/delete pvc volumes",
	}
	volumeListCmd := &cobra.Command{
		Use:     "list",
		Short:   "list volume",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.volumeFind()
		},
		PreRun: bindPFlags,
	}
	volumeDescribeCmd := &cobra.Command{
		Use:   "describe <volume>",
		Short: "describes a volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.volumeDescribe(args)
		},
		ValidArgsFunction: c.comp.VolumeListCompletion,
		PreRun:            bindPFlags,
	}
	volumeDeleteCmd := &cobra.Command{
		Use:     "delete <volume>",
		Aliases: []string{"rm", "destroy", "remove", "delete"},
		Short:   "delete a volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.volumeDelete(args)
		},
		ValidArgsFunction: c.comp.VolumeListCompletion,
		PreRun:            bindPFlags,
	}
	volumeManifestCmd := &cobra.Command{
		Use:   "manifest <volume>",
		Short: "print a manifest for a volume",
		Long:  "this is only useful for volumes which are not used in any k8s cluster. With the PersistenVolumeClaim given you can reuse it in a new cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.volumeManifest(args)
		},
		ValidArgsFunction: c.comp.VolumeListCompletion,
		PreRun:            bindPFlags,
	}
	volumeClusterInfoCmd := &cobra.Command{
		Use:   "clusterinfo",
		Short: "show storage cluster infos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.volumeClusterInfo()
		},
		PreRun: bindPFlags,
	}

	volumeCmd.AddCommand(volumeListCmd)
	volumeCmd.AddCommand(volumeDeleteCmd)
	volumeCmd.AddCommand(volumeDescribeCmd)
	volumeCmd.AddCommand(volumeManifestCmd)
	volumeCmd.AddCommand(volumeClusterInfoCmd)

	volumeListCmd.Flags().StringP("volumeid", "", "", "volumeid to filter [optional]")
	volumeListCmd.Flags().StringP("project", "", "", "project to filter [optional]")
	volumeListCmd.Flags().StringP("partition", "", "", "partition to filter [optional]")
	volumeListCmd.Flags().StringP("tenant", "", "", "tenant to filter [optional]")
	volumeListCmd.Flags().Bool("only-unbound", false, "show only unbound volumes that are not connected to any hosts, pv may be still present. [optional]")

	must(volumeListCmd.RegisterFlagCompletionFunc("project", c.comp.ProjectListCompletion))
	must(volumeListCmd.RegisterFlagCompletionFunc("partition", c.comp.PartitionListCompletion))
	must(volumeListCmd.RegisterFlagCompletionFunc("tenant", c.comp.TenantListCompletion))

	volumeManifestCmd.Flags().StringP("name", "", "restored-pv", "name of the PersistentVolume")
	volumeManifestCmd.Flags().StringP("namespace", "", "default", "namespace for the PersistentVolume")

	volumeClusterInfoCmd.Flags().StringP("partition", "", "", "partition to filter [optional]")
	must(volumeClusterInfoCmd.RegisterFlagCompletionFunc("partition", c.comp.PartitionListCompletion))

	return volumeCmd
}

func (c *config) volumeFind() error {
	if helper.AtLeastOneViperStringFlagGiven("volumeid", "project", "partition", "tenant") {
		params := volume.NewFindVolumesParams()
		ifr := &models.V1VolumeFindRequest{
			VolumeID:    helper.ViperString("volumeid"),
			ProjectID:   helper.ViperString("project"),
			PartitionID: helper.ViperString("partition"),
			TenantID:    helper.ViperString("tenant"),
		}
		params.SetBody(ifr)
		resp, err := c.cloud.Volume.FindVolumes(params, nil)
		if err != nil {
			return err
		}
		volumes := resp.Payload
		if viper.GetBool("only-unbound") {
			volumes = onlyUnboundVolumes(volumes)
		}
		return output.New().Print(volumes)
	}
	resp, err := c.cloud.Volume.ListVolumes(nil, nil)
	if err != nil {
		return err
	}
	volumes := resp.Payload
	if viper.GetBool("only-unbound") {
		volumes = onlyUnboundVolumes(volumes)
	}
	return output.New().Print(volumes)
}

func onlyUnboundVolumes(volumes []*models.V1VolumeResponse) (result []*models.V1VolumeResponse) {
	for _, v := range volumes {
		if len(v.ConnectedHosts) > 0 {
			continue
		}
		v := v
		result = append(result, v)
	}
	return result
}

func (c *config) volumeDescribe(args []string) error {
	vol, err := c.getVolumeFromArgs(args)
	if err != nil {
		return err
	}

	resp, err := c.cloud.Volume.GetVolume(volume.NewGetVolumeParams().WithID(*vol.VolumeID), nil)
	if err != nil {
		return err
	}

	return output.New().Print(resp.Payload)
}

func (c *config) volumeDelete(args []string) error {
	vol, err := c.getVolumeFromArgs(args)
	if err != nil {
		return err
	}

	if len(vol.ConnectedHosts) > 0 {
		return fmt.Errorf("volume is still connected to this node:%s", vol.ConnectedHosts)
	}

	if !viper.GetBool("yes-i-really-mean-it") {
		fmt.Printf(`
delete volume: %q, all data will be lost forever.
If used in cronjob for example, volume might not be connected now, but required at a later point in time.
`, *vol.VolumeID)
		err = helper.Prompt("Are you sure? (y/n)", "y")
		if err != nil {
			return err
		}
	}

	resp, err := c.cloud.Volume.DeleteVolume(volume.NewDeleteVolumeParams().WithID(*vol.VolumeID), nil)
	if err != nil {
		return err
	}

	return output.New().Print(resp.Payload)
}

func (c *config) volumeClusterInfo() error {
	params := volume.NewClusterInfoParams().WithPartitionid(helper.ViperString("partition"))
	resp, err := c.cloud.Volume.ClusterInfo(params, nil)
	if err != nil {
		return err
	}
	return output.New().Print(resp.Payload)
}

func (c *config) volumeManifest(args []string) error {
	volume, err := c.getVolumeFromArgs(args)
	if err != nil {
		return err
	}
	name := viper.GetString("name")
	namespace := viper.GetString("namespace")

	return output.VolumeManifest(*volume, name, namespace)
}
func (c *config) getVolumeFromArgs(args []string) (*models.V1VolumeResponse, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("no volume given")
	}

	volumeID := args[0]
	params := volume.NewFindVolumesParams()
	ifr := &models.V1VolumeFindRequest{
		VolumeID: &volumeID,
	}
	params.SetBody(ifr)
	resp, err := c.cloud.Volume.FindVolumes(params, nil)
	if err != nil {
		return nil, err
	}
	if len(resp.Payload) < 1 {
		return nil, fmt.Errorf("no volume for id:%s found", volumeID)
	}
	if len(resp.Payload) > 1 {
		return nil, fmt.Errorf("more than one volume for id:%s found", volumeID)
	}
	return resp.Payload[0], nil
}
