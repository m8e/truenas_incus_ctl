package cmd

import (
	"errors"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Short:   "Edit or list snapshots on a remote or local machine",
	Aliases: []string{"snap"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var snapshotCloneCmd = &cobra.Command{
	Use:   "clone <dataset>@<snapshot>",
	Short: "clone snapshot of ZFS dataset",
	Args:  cobra.ExactArgs(2),
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create <dataset>@<snapshot>...",
	Short: "Take a snapshot of dataset, possibly recursive",
	Args:  cobra.MinimumNArgs(1),
}

var snapshotDeleteCmd = &cobra.Command{
	Use:     "delete <dataset>@<snapshot>...",
	Short:   "Delete a snapshot of dataset, possibly recursive",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var snapshotListCmd = &cobra.Command{
	Use:     "list [<dataset>][@<snapshot>]...",
	Short:   "List all snapshots",
	Aliases: []string{"ls"},
}

var snapshotRenameCmd = &cobra.Command{
	Use:     "rename <old dataset>@<old snapshot> <new snapshot>",
	Short:   "Rename a ZFS snapshot",
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"mv"},
}

var snapshotRollbackCmd = &cobra.Command{
	Use:   "rollback <old dataset>@<old snapshot>",
	Short: "Rollback to a given snapshot",
	Args:  cobra.MinimumNArgs(1),
}

var g_snapshotListEnums map[string][]string

func init() {
	snapshotCloneCmd.RunE = WrapCommandFunc(cloneSnapshot)
	snapshotCreateCmd.RunE = WrapCommandFunc(createSnapshot)
	snapshotDeleteCmd.RunE = WrapCommandFunc(deleteOrRollbackSnapshot)
	snapshotListCmd.RunE = WrapCommandFunc(listSnapshot)
	snapshotRenameCmd.RunE = WrapCommandFunc(renameSnapshot)
	snapshotRollbackCmd.RunE = WrapCommandFunc(deleteOrRollbackSnapshot)

	snapshotCreateCmd.Flags().BoolP("delete", "d", false, "Delete snapshot if it exists already")
	snapshotCreateCmd.Flags().BoolP("recursive", "r", false, "")
	snapshotCreateCmd.Flags().String("exclude", "", "List of datasets to exclude")
	snapshotCreateCmd.Flags().StringP("option", "o", "", "Specify property=value,...")
	snapshotCreateCmd.Flags().Bool("suspend-vms", false, "")
	snapshotCreateCmd.Flags().Bool("vmware-sync", false, "")

	snapshotDeleteCmd.Flags().BoolP("recursive", "r", false, "recursively delete children")
	snapshotDeleteCmd.Flags().Bool("defer", false, "defer the deletion of snapshot")

	snapshotListCmd.Flags().BoolP("recursive", "r", false, "")
	snapshotListCmd.Flags().BoolP("user-properties", "u", false, "Include user-properties")
	snapshotListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	snapshotListCmd.Flags().BoolP("no-headers", "c", false, "Equivalent to --format=compact. More easily parsed by scripts")
	snapshotListCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+
		AddFlagsEnum(&g_snapshotListEnums, "format", []string{"csv", "json", "table", "compact"}))
	snapshotListCmd.Flags().StringP("output", "o", "", "Output property list")
	snapshotListCmd.Flags().BoolP("parsable", "p", false, "Show raw values instead of the already parsed values")
	snapshotListCmd.Flags().Bool("all", false, "Output all properties")

	snapshotRollbackCmd.Flags().BoolP("force", "f", false, "force unmount of any clones")
	snapshotRollbackCmd.Flags().BoolP("recursive", "r", false, "destroy any snapshots and bookmarks more recent than the one specified")
	snapshotRollbackCmd.Flags().BoolP("recursive-clones", "R", false, "like recursive, but also destroy any clones")
	snapshotRollbackCmd.Flags().Bool("recursive-rollback", false, "perform a completem recursive rollback of each child snapshots.\n"+
		"If any child does not have specified snapshot, this operation will fail.")

	snapshotCmd.AddCommand(snapshotCloneCmd)
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRenameCmd)
	snapshotCmd.AddCommand(snapshotRollbackCmd)
	rootCmd.AddCommand(snapshotCmd)
}

func cloneSnapshot(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	outMap := make(map[string]interface{})
	outMap["snapshot"] = args[0]
	outMap["dataset_dst"] = args[1]
	//outMap["dataset_properties"] = make(map[string]interface{})

	params := []interface{}{outMap}
	DebugJson(params)

	out, err := core.ApiCall(api, "zfs.snapshot.clone", defaultCallTimeout, params)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func createSnapshot(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, false, nil)
	datasetList := make([]string, len(args), len(args))
	nameList := make([]string, len(args), len(args))

	for i := 0; i < len(args); i++ {
		snapshot := args[i]
		datasetLen := strings.Index(snapshot, "@")
		if datasetLen <= 0 || datasetLen == len(snapshot)-1 {
			return errors.New("No dataset name was found in snapshot specifier.\nExpected <datasetname>@<snapshotname>.")
		}
		if snapshot[0] == '/' {
			return errors.New("Dataset names must not start with '/'.")
		}

		dataset := snapshot[0:datasetLen]
		snapshotIsolated := snapshot[datasetLen+1:]

		datasetList[i] = dataset
		nameList[i] = snapshotIsolated
	}

	outMap := make(map[string]interface{})
	outMap["dataset"] = datasetList[0]
	outMap["name"] = nameList[0]

	MaybeCopyProperty(outMap, options.allFlags, "recursive")
	MaybeCopyProperty(outMap, options.usedFlags, "suspend_vms")
	MaybeCopyProperty(outMap, options.usedFlags, "vmware_sync")

	if excludeStr := options.allFlags["exclude"]; excludeStr != "" {
		outMap["exclude"] = strings.Split(excludeStr, ",")
	}

	// TODO: naming_schema

	outProps := make(map[string]interface{})
	_ = WriteKvArrayToMap(outProps, ConvertParamsStringToKvArray(options.allFlags["option"]), nil)
	outMap["properties"] = outProps

	params := []interface{}{outMap}

	cmd.SilenceUsage = true

	if core.IsStringTrue(options.allFlags, "delete") {
		delMap := make(map[string]interface{})
		delMap["recursive"] = true
		delObjRemap := map[string][]interface{}{"": core.ToAnyArray(args)}
		_, _, _ = MaybeBulkApiCall(api, "zfs.snapshot.delete", 10, []interface{}{args[0], delMap}, delObjRemap, true)
	}

	objRemap := map[string][]interface{}{"dataset": core.ToAnyArray(datasetList), "name": core.ToAnyArray(nameList)}
	out, _, err := MaybeBulkApiCall(api, "zfs.snapshot.create", 10, params, objRemap, false)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func deleteOrRollbackSnapshot(cmd *cobra.Command, api core.Session, args []string) error {
	cmdType := strings.Split(cmd.Use, " ")[0]
	if cmdType != "delete" && cmdType != "rollback" {
		return errors.New("cmdType was not delete or rollback")
	}

	snapshots := args
	for i := 0; i < len(args); i++ {
		datasetLen := strings.Index(snapshots[i], "@")
		if datasetLen <= 0 {
			return errors.New("No dataset name was found in snapshot specifier.\nExpected <datasetname>@<snapshotname>.")
		}
	}

	options, _ := GetCobraFlags(cmd, false, nil)
	params := BuildNameStrAndPropertiesJson(options, snapshots[0])

	cmd.SilenceUsage = true

	objRemap := map[string][]interface{}{"": core.ToAnyArray(snapshots)}
	out, _, err := MaybeBulkApiCall(api, "zfs.snapshot."+cmdType, 10, params, objRemap, false)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func renameSnapshot(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	srcType, source := core.IdentifyObject(args[0])
	dstType, dest := core.IdentifyObject(args[1])

	if srcType != "snapshot" {
		return errors.New("\"" + source + "\" is not a snapshot")
	}

	dsName := source[0:strings.Index(source, "@")]

	if dstType != "snapshot" {
		dest = dsName + "@" + dest
	} else if !strings.HasPrefix(dest, dsName+"@") {
		return errors.New(
			"The destination snapshot does not share the same dataset as the source.\n" +
				"Try leaving out the dataset name in the destination.")
	}

	params := []interface{}{source, dest}
	DebugJson(params)

	out, err := core.ApiCall(api, "zfs.snapshot.rename", defaultCallTimeout, params)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func listSnapshot(cmd *cobra.Command, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, false, g_snapshotListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getSnapshotListTypes(args)
	if err != nil {
		return err
	}

	// `zfs list` will "recurse" if no names are specified.
	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsStringTrue(options.allFlags, "parsable")),
		shouldGetAllProps:  core.IsStringTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      len(args) == 0 || core.IsStringTrue(options.allFlags, "recursive"),
	}

	response, err := QueryApi(api, "zfs.snapshot", args, idTypes, properties, extras)
	if err != nil {
		return err
	}

	snapshots := GetListFromQueryResponse(&response)
	//LowerCaseValuesFromEnums(snapshots, g_snapshotCreateUpdateEnums)

	required := []string{"name"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(snapshots, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, "snapshots", columnsList, snapshots)
	PrintTable(api, str)
	return err
}

func getSnapshotListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t, value := core.IdentifyObject(args[i])
		if t == "id" || t == "share" {
			return nil, errors.New("querying snapshots based on mount point is not yet supported")
		} else if t == "snapshot" {
			t = "name"
		} else if t == "snapshot_only" {
			t = "snapshot_name"
		} else if t != "dataset" && t != "pool" {
			return nil, errors.New("Unrecognised namespec \"" + args[i] + "\"")
		}
		typeList[i] = t
		args[i] = value
	}

	return typeList, nil
}
