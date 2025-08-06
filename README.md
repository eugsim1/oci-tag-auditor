
```markdown
# OCI Resource Tag Auditor

A Go utility to audit Oracle Cloud Infrastructure (OCI) resources across all regions, focusing on tag compliance and resource metadata.


## Features

- **Comprehensive Resource Discovery**: Lists all OCI resources across configured regions
- **Tag Compliance Reporting**:
  - Identifies resources missing defined tags (`-missing-tags` flag)
  - Identifies resources missing `CreatedBy` tag (`-no-owner` flag)
- **Time Tracking**:
  - Accurate creation timestamps (UTC)
  - Days since resource creation
- **Parallel Processing**: Concurrent scanning of multiple regions
- **Flexible Output**: Generates CSV reports with configurable detail levels

## Prerequisites

- Go 1.16+ ([installation guide](https://golang.org/doc/install))
- OCI CLI configured with proper permissions
- OCI Go SDK dependencies

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/eugsim1/oci-tag-auditor.git
   cd oci-tag-auditor
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the binary:
   ```bash
   go build -o oci-tag-auditor
   ```

## Configuration

1. Create a `config_path.txt` file containing the path to your OCI config file
   ```
   /path/to/your/oci/config
   ```

2. Ensure your OCI config file has:
   - A DEFAULT profile with home region credentials
   - Additional profiles for each region to audit

Example config structure:
```ini
[DEFAULT]
user=ocid1.user.oc1..<unique_ID>
fingerprint=<your_fingerprint>
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy.oc1..<unique_ID>
region=us-phoenix-1

[us-phoenix-1]
user=ocid1.user.oc1..<unique_ID>
fingerprint=<your_fingerprint>
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy.oc1..<unique_ID>
region=us-phoenix-1

[eu-frankfurt-1]
user=ocid1.user.oc1..<unique_ID>
fingerprint=<your_fingerprint>
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy.oc1..<unique_ID>
region=eu-frankfurt-1
```

## Usage

```bash
./oci-tag-auditor [flags]
```

### Available Flags

| Flag          | Description                                      |
|---------------|--------------------------------------------------|
| `-missing-tags` | Generate report for resources missing defined tags |
| `-no-owner`    | Generate report for resources missing CreatedBy tag |

### Examples

1. Basic audit (main report only):
   ```bash
   ./oci-tag-auditor
   ```

2. Audit with missing tags report:
   ```bash
   ./oci-tag-auditor -missing-tags
   ```

3. Full audit with all reports:
   ```bash
   ./oci-tag-auditor -missing-tags -no-owner
   ```

## Output Files

The utility creates CSV reports in the `data/` directory with timestamped filenames:

1. **Main Report**: `<region>_resources_<timestamp>.csv`
   - Contains all discovered resources with complete metadata

2. **Missing Tags Report**: `<region>_missing_tags_<timestamp>.csv` (with `-missing-tags` flag)
   - Contains resources with no defined tags

3. **No Owner Report**: `<region>_no_owner_<timestamp>.csv` (with `-no-owner` flag)
   - Contains resources missing the CreatedBy tag

### Report Columns

All reports include these columns:

1. Region
2. Display Name
3. Resource Type
4. Identifier (OCID)
5. Compartment ID
6. Lifecycle State
7. Time Created (UTC) - Format: `YYYY-MM-DD HH:MM:SS`
8. Days Since Creation
9. Availability Domain
10. Defined Tags (JSON format)
11. Freeform Tags (key=value pairs)

## Sample Output

```csv
Region,Display Name,Resource Type,Identifier,Compartment ID,Lifecycle State,Time Created (UTC),Days Since Creation,Availability Domain,Defined Tags,Freeform Tags
us-phoenix-1,my-vm,Instance,ocid1.instance.oc1..xxxxx,ocid1.compartment.oc1..xxxxx,RUNNING,2023-05-15 14:30:00,180,AD-1,"{""Oracle-Tags"":{""CreatedBy"":""john.doe""}}","environment=prod,owner=team-a"
us-phoenix-1,my-db,AutonomousDatabase,ocid1.autonomousdatabase.oc1..xxxxx,ocid1.compartment.oc1..xxxxx,AVAILABLE,2023-07-20 08:15:00,120,,"",""
```

## Troubleshooting

1. **Authentication Errors**:
   - Verify your OCI config file path in `config_path.txt`
   - Ensure your API key has proper permissions

2. **Missing Dependencies**:
   ```bash
   go get github.com/oracle/oci-go-sdk/v65/common
   go get github.com/oracle/oci-go-sdk/v65/identity
   go get github.com/oracle/oci-go-sdk/v65/resourcesearch
   go get gopkg.in/ini.v1
   ```

3. **Permission Issues**:
   - Ensure the `data/` directory is writable
   - Verify your OCI user has proper permissions to list resources

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/your-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin feature/your-feature`)
5. Create a new Pull Request
```

