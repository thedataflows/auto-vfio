# auto-vfio

Tool to automate the setup of VFIO passthrough on Linux systems.

## Usage

```properties
Usage: auto-vfio <command> [flags]

Automate the setup of VFIO passthrough on Linux systems

Flags:
  -h, --help                          Show context-sensitive help.
  -c, --config-file="default.yaml"    Config file location. Supported formats: .json, .yaml, .yml, .toml
  -l, --log-level="info"              Logging level. One of: trace, debug, info, warn, error, fatal, panic

Commands:
  list (l) [flags]
    List IOMMU groups and PCI devices

  rebind (r) --bus=bus-address1,... [flags]
    Rebind a device from its driver to vfio-pci

  version [flags]
    Show version information and exit

Run "auto-vfio <command> --help" for more information on a command.
```

### Rebind devices

```properties
Usage: auto-vfio rebind (r) --bus=bus-address1,... [flags]

Rebind a device from its driver to vfio-pci

Flags:
  -h, --help                          Show context-sensitive help.
  -c, --config-file="default.yaml"    Config file location. Supported formats: .json, .yaml, .yml, .toml
  -l, --log-level="info"              Logging level. One of: trace, debug, info, warn, error, fatal, panic

  -b, --bus=bus-address1,...          Comma separated lisf of Bus addresses. Use 'list' command to get them. Example: 0000:07:00.0,0000:07:00.1
  -p, --persist                       Persist binding to vfio-pci across reboots
```

### List devices

Output is similar to `lspci -nnk` but with additional information about IOMMU groups. Using <https://github.com/TimRots/gutil-linux> for interpreting PCI devices and vendors.

```properties
auto-vfio list

Usage: auto-vfio list (c) [flags]

List IOMMU groups and PCI devices

Flags:
  -h, --help                          Show context-sensitive help.
  -c, --config-file="default.yaml"    Config file location. Supported formats: .json, .yaml, .yml, .toml
  -l, --log-level="info"              Logging level. One of: trace, debug, info, warn, error, fatal, panic

      --tree                          Hierarchical output
  -o, --output-format=""              Output format. One of: json, yaml, xml, toml, props, shell, csv, tsv,
  -y, --yq=STRING                     YQ expression to apply to the output. Ignored if output format is not specified
```

- Filtering devices with <https://mikefarah.gitbook.io/yq> expressions:

  ```bash
  ./auto-vfio list \
      --tree \
      --yq 'with_entries(select(.value[] | .DeviceClass | test("VGA")))'
  ```

- **Note**: for `csv`/`tsv`, when filtering with yq, the resulting data must be flatened.

  For example, if you want to filter csv/tsv and pretty print only some columns:

  ```bash
  ./auto-vfio list \
    --output-format=tsv \
    --yq '[.[] | select(.DeviceClass | test("VGA")) | {"Group": .IommuGroup, "Bus": .Bus, "DeviceName": .DeviceName}]' | \
      column -t -s $'\t'
  ```

  ```properties
  Group  Bus           DeviceName
  17     0000:01:00.0  GA102 [GeForce RTX 3080 12GB]
  20     0000:07:00.0  AD107 [GeForce RTX 4060]
  ```

## Develop

- Build: `go build .`
- Run: `go run .`
- Test: `go test ./...`

## License

[MIT](LICENSE)
