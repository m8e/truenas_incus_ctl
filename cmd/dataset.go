package cmd

import (
	//"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Edit or list datasets/zvols and their shares on a remote or local machine",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var datasetCreateCmd = &cobra.Command{
	Use:   "create <dataset>...",
	Short: "Creates a dataset/zvol.",
	Args:  cobra.MinimumNArgs(1),
}

var datasetUpdateCmd = &cobra.Command{
	Use:     "update <dataset>...",
	Short:   "Updates an existing dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"set"},
}

var datasetDeleteCmd = &cobra.Command{
	Use:     "delete <dataset>...",
	Short:   "Deletes a dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var datasetListCmd = &cobra.Command{
	Use:     "list [dataset]...",
	Short:   "Prints a table of all datasets/zvols, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
}

var datasetPromoteCmd = &cobra.Command{
	Use:   "promote <dataset>...",
	Short: "Promote a clone dataset to no longer depend on the origin snapshot.",
	Args:  cobra.MinimumNArgs(1),
}

var datasetRenameCmd = &cobra.Command{
	Use:   "rename <old dataset>[@<old snapshot>] <new dataset|new snapshot>",
	Short: "Rename a ZFS dataset",
	Long: `Renames the given dataset. The new target can be located anywhere in the ZFS hierarchy, with the exception of snapshots.
Snapshots can only be re‐named within the parent file system or volume.
When renaming a snapshot, the parent file system of the snapshot does not need to be specified as part of the second argument.
Renamed file systems can inherit new mount points, in which case they are unmounted and remounted at the new mount point.`,
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"mv"},
}

var g_compressionEnum = [...]string{
	"on", "off", "gzip",
	"gzip-1", "gzip-9",
	"lz4", "lzjb", "zle", "zstd",
	"zstd-1", "zstd-2", "zstd-3", "zstd-4", "zstd-5", "zstd-6", "zstd-7", "zstd-8", "zstd-9", "zstd-10",
	"zstd-11", "zstd-12", "zstd-13", "zstd-14", "zstd-15", "zstd-16", "zstd-17", "zstd-18", "zstd-19",
	"zstd-fast",
	"zstd-fast-1", "zstd-fast-2", "zstd-fast-3", "zstd-fast-4", "zstd-fast-5", "zstd-fast-6", "zstd-fast-7", "zstd-fast-8", "zstd-fast-9",
	"zstd-fast-10", "zstd-fast-20", "zstd-fast-30", "zstd-fast-40", "zstd-fast-50", "zstd-fast-60", "zstd-fast-70", "zstd-fast-80", "zstd-fast-90",
	"zstd-fast-100", "zstd-fast-500", "zstd-fast-1000",
}

var g_datasetCreateUpdateEnums map[string][]string
var g_datasetListEnums map[string][]string

func init() {
	datasetCreateCmd.RunE = WrapCommandFunc(createOrUpdateDataset)
	datasetUpdateCmd.RunE = WrapCommandFunc(createOrUpdateDataset)
	datasetDeleteCmd.RunE = WrapCommandFunc(deleteDataset)
	datasetListCmd.RunE = WrapCommandFunc(listDataset)
	datasetPromoteCmd.RunE = WrapCommandFunc(promoteDataset)
	datasetRenameCmd.RunE = WrapCommandFunc(renameDataset)

	createUpdateCmds := []*cobra.Command{datasetCreateCmd, datasetUpdateCmd}
	for _, cmd := range createUpdateCmds {
		cmd.Flags().String("comments", "", "User defined comments")
		cmd.Flags().String("managedby", "truenas_incus_ctl", "Manager of this dataset, must not be empty")
		cmd.Flags().String("recordsize", "", "")
		cmd.Flags().String("sync", "standard", "Controls the behavior of synchronous requests "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "sync", []string{"standard", "always", "disabled"}))
		cmd.Flags().String("snapdir", "hidden", "Controls whether the .zfs directory is disabled, hidden or visible "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "snapdir", []string{"disabled", "hidden", "visible"}))
		cmd.Flags().String("compression", "off", "Controls the compression algorithm used for this dataset\n"+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "compression", g_compressionEnum[:]))
		cmd.Flags().String("atime", "inherit", "Controls whether the access time for files is updated when they are read "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "atime", []string{"inherit", "on", "off"}))
		//cmd.Flags().String("relatime", "inherit", "Controls whether the access time for files is updated periodically "+
		//AddFlagsEnum(&g_datasetCreateUpdateEnums, "relatime", []string{"inherit", "on", "off"}))
		cmd.Flags().String("exec", "inherit", "Controls whether processes can be executed from within this file system "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "exec", []string{"inherit", "on", "off"}))
		cmd.Flags().String("acltype", "inherit", "Controls whether ACLs are enabled and if so what type of ACL to use "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "acltype", []string{"inherit", "posix", "nfsv4", "off"}))
		cmd.Flags().String("aclmode", "inherit", "Controls how an ACL is modified during chmod(2) and how inherited ACEs are modified by the file creation mode "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "aclmode", []string{"inherit", "passthrough", "restricted", "discard"}))
		cmd.Flags().String("deduplication", "inherit", ""+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "deduplication", []string{"inherit", "on", "verify", "off"}))
		cmd.Flags().String("checksum", "inherit", ""+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "checksum", []string{"inherit", "on", "off", "fletcher2", "fletcher4", "sha256", "sha512", "skein", "edonr", "blake3"}))
		cmd.Flags().String("readonly", "inherit", ""+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "readonly", []string{"inherit", "on", "off"}))
		cmd.Flags().String("casesensitivity", "inherit", ""+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "casesensitivity", []string{"inherit", "sensitive", "insensitive"}))
		cmd.Flags().String("share-type", "inherit", ""+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "share_type", []string{"inherit", "generic", "multiprotocol", "nfs", "smb", "apps"}))
		//cmd.Flags().String("xattr", "inherit", "Controls whether extended attributes are enabled for this file system "+
		//	AddFlagsEnum(&g_datasetCreateUpdateEnums, "xattr", []string{"inherit", "on", "off", "dir"})) // 'sa' should be "on"
		//cmd.Flags().String("encryption-options", "", "")
		//cmd.Flags().Bool("encryption", false, "")
		//cmd.Flags().Bool("inherit-encryption", true, "")
		cmd.Flags().String("quota", "0", "")
		cmd.Flags().Int("quota-warning", 0, "Percentage (1-100 or 0)")
		cmd.Flags().Int("quota-critical", 0, "Percentage (1-100 or 0)")
		cmd.Flags().String("refquota", "0", "")
		cmd.Flags().Int("refquota-warning", 0, "Percentage (1-100 or 0)")
		cmd.Flags().Int("refquota-critical", 0, "Percentage (1-100 or 0)")
		cmd.Flags().String("reservation", "0", "")
		cmd.Flags().String("refreservation", "0", "")
		cmd.Flags().String("special-small-block-size", "0", "")
		cmd.Flags().Int("copies", 0, "")
		cmd.Flags().BoolP("create-parents", "p", false, "Creates all the non-existing parent datasets")
		cmd.Flags().StringP("user-props", "u", "", "Sets the specified properties")
		cmd.Flags().StringP("option", "o", "", "Specify property=value,...")
		cmd.Flags().Bool("allow-shrinking", false, "By default, shrinking a volume to a smaller size is not permitted. This flag disables this check.")
		cmd.Flags().StringP("volsize", "V", "0", "Creates a volume of the given size instead of a filesystem, should be a multiple of the block size.")
		cmd.Flags().StringP("volblocksize", "b", "512", "Volume block size "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "volblocksize", []string{"512", "1K", "2K", "4K", "8K", "16K", "32K", "64K", "128K"}))
		cmd.Flags().BoolP("sparse", "s", false, "Creates a sparse volume with no reservation")
		cmd.Flags().Bool("force-size", false, "")
		cmd.Flags().String("snapdev", "hidden", "Controls whether the volume snapshot devices are hidden or visible "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "snapdev", []string{"hidden", "visible"}))
	}

	datasetUpdateCmd.Flags().Bool("create", false, "If a dataset doesn't exist, create it. Off by default.")

	g_datasetCreateUpdateEnums["type"] = []string{"volume", "filesystem"}

	datasetDeleteCmd.Flags().BoolP("recursive", "r", false, "Also delete/destroy all children datasets. When the root dataset is specified,\n"+
		"it will destroy all the children of the root dataset present leaving root dataset intact")
	datasetDeleteCmd.Flags().BoolP("force", "f", false, "Force delete busy datasets")
	datasetDeleteCmd.Flags().Bool("no-smart-timeout", false, "Disable performing a recursive list on the dataset to determine a suitable deletion timeout")

	datasetListCmd.Flags().BoolP("recursive", "r", false, "Retrieves properties for children")
	datasetListCmd.Flags().BoolP("user-properties", "u", false, "Include user-properties")
	datasetListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	datasetListCmd.Flags().BoolP("no-headers", "c", false, "Equivalent to --format=compact. More easily parsed by scripts")
	datasetListCmd.Flags().String("format", "table", "Output table format "+
		AddFlagsEnum(&g_datasetListEnums, "format", []string{"csv", "json", "table", "compact"}))
	datasetListCmd.Flags().StringP("output", "o", "", "Output property list")
	datasetListCmd.Flags().BoolP("parsable", "p", false, "Show raw values instead of the already parsed values")
	datasetListCmd.Flags().BoolP("all", "a", false, "Output all properties")
	datasetListCmd.Flags().StringP("source", "s", "default", "A comma-separated list of sources to display.\n"+
		"Those properties coming from a source other than those in this list are ignored.\n"+
		"Each source must be one of the following: local, default, inherited, temporary, received, or none.\n"+
		"The default value is all sources.")

	datasetRenameCmd.Flags().BoolP("update-shares", "s", false, "Will update any shares as part of rename")

	datasetCmd.AddCommand(datasetCreateCmd)
	datasetCmd.AddCommand(datasetUpdateCmd)
	datasetCmd.AddCommand(datasetDeleteCmd)
	datasetCmd.AddCommand(datasetListCmd)
	datasetCmd.AddCommand(datasetPromoteCmd)
	datasetCmd.AddCommand(datasetRenameCmd)
	rootCmd.AddCommand(datasetCmd)
}

func createOrUpdateDataset(cmd *cobra.Command, api core.Session, args []string) error {
	cmdType := strings.Split(cmd.Use, " ")[0]
	if cmdType != "create" && cmdType != "update" {
		return errors.New("cmdType was not create or update")
	}

	options, err := GetCobraFlags(cmd, false, g_datasetCreateUpdateEnums)
	if err != nil {
		return err
	}

	specs := make([]string, len(args), len(args))
	types := make([]string, len(args), len(args)) // always "name" repeated
	for i, ds := range args {
		idType, spec := core.IdentifyObject(ds)
		if idType != "dataset" {
			return fmt.Errorf("dataset %s only operates on datasets (%s is a %s)", cmdType, spec, idType)
		}
		specs[i] = spec
		types[i] = "name" // always "name"
	}

	flagCreate := core.IsStringTrue(options.allFlags, "create")
	RemoveFlag(options, "create")

	allowShrinking := core.IsStringTrue(options.allFlags, "allow_shrinking")
	allowShrinking = allowShrinking
	RemoveFlag(options, "allow_shrinking")

	outMap := make(map[string]interface{})

	var userPropsStr string

	for propName, valueStr := range options.usedFlags {
		isProp := false
		switch propName {
		case "create_parents":
			outMap["create_ancestors"] = valueStr == "true"
		case "quota":
			fallthrough
		case "refquota":
			fallthrough
		case "reservation":
			fallthrough
		case "refreservation":
			fallthrough
		case "special-small-block-size":
			fallthrough
		case "volsize":
			size, err := core.ParseSizeString(valueStr)
			if err != nil {
				return errors.New("Failed to parse " + propName + ": " + err.Error())
			}
			if size < 0 {
				return errors.New("Failed to parse " + propName + ": negative numbers are not permitted")
			}
			outMap[propName] = size
		case "user_props":
			userPropsStr = valueStr
		case "option":
			kvArray := ConvertParamsStringToKvArray(valueStr)
			if err = WriteKvArrayToMap(outMap, kvArray, g_datasetCreateUpdateEnums); err != nil {
				return err
			}
		default:
			isProp = true
		}
		if isProp {
			value, err := ParseStringAndValidate(propName, valueStr, g_datasetCreateUpdateEnums)
			if err != nil {
				return err
			}
			outMap[propName] = value
		}
	}

	if userPropsStr != "" {
		kvParams := ConvertParamsStringToKvArray(userPropsStr)
		userPropsArr := make([]map[string]interface{}, 0)
		for i := 0; i < len(kvParams); i += 2 {
			value, err := ParseStringAndValidate(kvParams[i], kvParams[i+1], g_datasetCreateUpdateEnums)
			if err != nil {
				return err
			}
			m := make(map[string]interface{})
			m["key"] = kvParams[i]
			m["value"] = value
			userPropsArr = append(userPropsArr, m)
		}
		outMap["user_properties"] = userPropsArr
	}

	cmd.SilenceUsage = true

	var listToCreate []string
	var listToUpdate []string

	if cmdType == "create" {
		listToCreate = specs
	} else if len(specs) > 1 || flagCreate {
		extras := typeQueryParams{
			valueOrder:         BuildValueOrder(true),
			shouldGetAllProps:  false,
			shouldGetUserProps: false,
			shouldRecurse:      false,
		}
		response, err := QueryApi(api, "pool.dataset", specs, types, nil, extras)
		if err != nil {
			return err
		}

		listToCreate = make([]string, 0)
		listToUpdate = make([]string, 0)
		for _, spec := range specs {
			if _, exists := response.resultsMap[spec]; exists {
				listToUpdate = append(listToUpdate, spec)
			} else {
				if !flagCreate {
					return errors.New("Could not find dataset \"" + spec + "\".\n" +
						"Try passing -c or --create to create a dataset if it doesn't exist.")
				}
				listToCreate = append(listToCreate, spec)
			}
		}
	} else {
		listToUpdate = specs
	}

	if len(listToUpdate) > 0 {
		objRemap := map[string][]interface{}{"": core.ToAnyArray(listToUpdate)}
		out, _, err := MaybeBulkApiCall(api, "pool.dataset.update", 10, []interface{}{outMap}, objRemap, false)
		if err != nil {
			return err
		}
		DebugString(string(out))
	}

	if len(listToCreate) > 0 {
		if _, exists := outMap["volsize"]; exists {
			outMap["type"] = "VOLUME"
		} else {
			outMap["type"] = "FILESYSTEM"
		}

		objRemap := map[string][]interface{}{"name": core.ToAnyArray(listToCreate)}
		out, _, err := MaybeBulkApiCall(api, "pool.dataset.create", 10, []interface{}{outMap}, objRemap, false)
		if err != nil {
			return err
		}
		DebugString(string(out))
	}

	return nil
}

func deleteDataset(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	options, _ := GetCobraFlags(cmd, false, nil)
	timeout := int64(20)

	if core.IsStringTrue(options.allFlags, "no_smart_timeout") {
		RemoveFlag(options, "no_smart_timeout")
	} else if core.IsStringTrue(options.allFlags, "recursive") {
		extras := typeQueryParams{
			valueOrder:         BuildValueOrder(true),
			shouldGetAllProps:  false,
			shouldGetUserProps: false,
			shouldRecurse:      true,
		}
		response, err := QueryApi(api, "pool.dataset", args, core.StringRepeated("name", len(args)), []string{}, extras)
		if err != nil {
			return err
		}
		timeout = int64(10 + 10*len(response.resultsMap))
	}

	params := BuildNameStrAndPropertiesJson(options, args[0])

	objRemap := map[string][]interface{}{"": core.ToAnyArray(args)}
	out, _, err := MaybeBulkApiCall(api, "pool.dataset.delete", timeout, params, objRemap, false)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func listDataset(cmd *cobra.Command, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, false, g_datasetListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getDatasetListTypes(args)
	if err != nil {
		return err
	}

	// `zfs list` will "recurse" if no names are specified.
	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsStringTrue(options.allFlags, "parsable")),
		shouldGetAllProps:  core.IsStringTrue(options.allFlags, "all"),
		shouldGetUserProps: core.IsStringTrue(options.allFlags, "user_properties"),
		shouldRecurse:      len(args) == 0 || core.IsStringTrue(options.allFlags, "recursive"),
	}

	for _, prop := range properties {
		if strings.Index(prop, ":") >= 0 {
			extras.shouldGetUserProps = true
			break
		}
	}

	response, err := QueryApi(api, "pool.dataset", args, idTypes, properties, extras)
	if err != nil {
		return err
	}

	datasets := GetListFromQueryResponse(&response)
	LowerCaseValuesFromEnums(datasets, g_datasetCreateUpdateEnums)

	required := []string{"name"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(datasets, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, "datasets", columnsList, datasets)
	PrintTable(api, str)
	return err
}

func promoteDataset(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	params := []interface{}{args[0]}
	objRemap := map[string][]interface{}{"": core.ToAnyArray(args)}
	out, _, err := MaybeBulkApiCall(api, "pool.dataset.promote", 10, params, objRemap, false)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func renameDataset(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	options, _ := GetCobraFlags(cmd, false, nil)

	source := args[0]
	dest := args[1]

	outMap := make(map[string]interface{})
	outMap["new_name"] = dest

	params := []interface{}{source, outMap}
	DebugJson(params)

	out, err := core.ApiCall(api, "zfs.dataset.rename", defaultCallTimeout, params)
	if err != nil {
		return err
	}
	DebugString(string(out))

	// no point updating the share if we're renaming a snapshot.
	if core.IsStringTrue(options.allFlags, "update_shares") && !strings.Contains(source, "@") {
		idStr, found, err := LookupNfsIdByPath(api, "/mnt/"+source, nil)
		if err != nil {
			return err
		}
		if !found {
			fmt.Println("INFO: this dataset did not appear to have a share")
			return nil
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("Error updating share for dataset \"%s\", nfs id \"%s\": %v", dest, idStr, err)
		}

		pathMap := make(map[string]interface{})
		pathMap["path"] = "/mnt/" + dest
		nfsParams := []interface{}{id, pathMap}

		DebugJson(nfsParams)

		out, err = core.ApiCall(api, "sharing.nfs.update", defaultCallTimeout, nfsParams)
		if err != nil {
			return err
		}
		DebugString(string(out))
	}

	return err
}

func getDatasetListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t, value := core.IdentifyObject(args[i])
		if t == "id" || t == "share" {
			return nil, errors.New("querying datasets based on mount point is not yet supported")
		} else if t == "snapshot" || t == "snapshot_only" {
			return nil, errors.New("querying datasets based on shapshot is not yet supported")
		} else if t == "dataset" {
			t = "name"
		} else if t != "pool" {
			return nil, errors.New("Unrecognised namespec \"" + args[i] + "\"")
		}
		typeList[i] = t
		args[i] = value
	}

	return typeList, nil
}
