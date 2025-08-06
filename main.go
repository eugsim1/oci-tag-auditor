package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/resourcesearch"
	"gopkg.in/ini.v1"
)

var (
	createMissingTagsFile bool
	createNoOwnerFile     bool
)

func init() {
	flag.BoolVar(&createMissingTagsFile, "missing-tags", false, "Create a separate file for resources with missing defined tags")
	flag.BoolVar(&createNoOwnerFile, "no-owner", false, "Create a separate file for resources with missing CreatedBy tag")
	flag.Parse()
}

func DefinedTagsToString(dt map[string]map[string]interface{}) string {
	bytes, err := json.Marshal(dt)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func FreeformTagsToString(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	for k, v := range tags {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func hasCreatedByTag(definedTags map[string]map[string]interface{}) bool {
	if len(definedTags) == 0 {
		return false
	}

	for _, namespace := range definedTags {
		for key, value := range namespace {
			if strings.EqualFold(key, "CreatedBy") {
				if strVal, ok := value.(string); ok && strVal != "" {
					return true
				}
			}
		}
	}
	return false
}

func formatTimeCreated(sdkTime *common.SDKTime) (string, string) {
	if sdkTime == nil {
		return "N/A", "N/A"
	}

	createdTime := sdkTime.Time
	formattedTime := createdTime.UTC().Format("2006-01-02 15:04:05")
	days := int(time.Since(createdTime).Hours() / 24)
	return formattedTime, fmt.Sprintf("%d", days)
}

func GetHomeRegionKeyFromDefaultConfig(ctx context.Context) (string, error) {

	configFilePath, err := ReadFirstLine("config_path.txt")
	if err != nil {
		log.Fatalf("Error reading config path: %v", err)
	}
	log.Printf("Using config file: %s", configFilePath)

	profileName := "DEFAULT"

	provider, err := common.ConfigurationProviderFromFileWithProfile(configFilePath, profileName, "")
	if err != nil {
		return "", fmt.Errorf("failed to create configuration provider: %w", err)
	}

	idClient, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return "", fmt.Errorf("failed to create IdentityClient: %w", err)
	}

	tenancyID, err := provider.TenancyOCID()
	if err != nil {
		return "", fmt.Errorf("failed to read tenancy OCID: %w", err)
	}

	req := identity.GetTenancyRequest{TenancyId: &tenancyID}
	resp, err := idClient.GetTenancy(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GetTenancy call failed: %w", err)
	}

	if resp.Tenancy.HomeRegionKey == nil {
		return "", fmt.Errorf("tenancy response missing HomeRegionKey")
	}
	return *resp.Tenancy.HomeRegionKey, nil
}

func ReadFirstLine(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return "", fmt.Errorf("file is empty")
}

func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func ExecuteFullSearch(configPath, section, query string) {
	ctx := context.Background()
	timestamp := time.Now().UTC().Format("20060102_150405")

	// Initialize OCI client
	configProvider, err := common.ConfigurationProviderFromFileWithProfile(configPath, section, "")
	if err != nil {
		log.Printf("Error creating configuration provider for %s: %v", section, err)
		return
	}

	client, err := resourcesearch.NewResourceSearchClientWithConfigurationProvider(configProvider)
	if err != nil {
		log.Printf("Error creating client for %s: %v", section, err)
		return
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
		return
	}

	// Create main report file
	mainReportFile, err := os.Create(fmt.Sprintf("data/%s_resources_%s.csv", section, timestamp))
	if err != nil {
		log.Printf("Error creating main report file: %v", err)
		return
	}
	defer mainReportFile.Close()

	mainWriter := csv.NewWriter(mainReportFile)
	defer mainWriter.Flush()

	// Initialize optional report files
	var (
		missingTagsFile, noOwnerFile     *os.File
		missingTagsWriter, noOwnerWriter *csv.Writer
	)

	if createMissingTagsFile {
		missingTagsFile, err = os.Create(fmt.Sprintf("data/%s_missing_tags_%s.csv", section, timestamp))
		if err != nil {
			log.Printf("Error creating missing tags file: %v", err)
			return
		}
		defer missingTagsFile.Close()
		missingTagsWriter = csv.NewWriter(missingTagsFile)
		defer missingTagsWriter.Flush()
	}

	if createNoOwnerFile {
		noOwnerFile, err = os.Create(fmt.Sprintf("data/%s_no_owner_%s.csv", section, timestamp))
		if err != nil {
			log.Printf("Error creating no owner file: %v", err)
			return
		}
		defer noOwnerFile.Close()
		noOwnerWriter = csv.NewWriter(noOwnerFile)
		defer noOwnerWriter.Flush()
	}

	// Write CSV headers
	headers := []string{
		"Region",
		"Display Name",
		"Resource Type",
		"Identifier",
		"Compartment ID",
		"Lifecycle State",
		"Time Created (UTC)",
		"Days Since Creation",
		"Availability Domain",
		"Defined Tags",
		"Freeform Tags",
	}

	if err := mainWriter.Write(headers); err != nil {
		log.Printf("Error writing main report header: %v", err)
		return
	}

	if createMissingTagsFile {
		if err := missingTagsWriter.Write(headers); err != nil {
			log.Printf("Error writing missing tags header: %v", err)
			return
		}
	}

	if createNoOwnerFile {
		if err := noOwnerWriter.Write(headers); err != nil {
			log.Printf("Error writing no owner header: %v", err)
			return
		}
	}

	// Perform resource search
	request := resourcesearch.SearchResourcesRequest{
		SearchDetails: resourcesearch.StructuredSearchDetails{
			Query: common.String(query),
		},
		Limit: common.Int(1000),
	}

	var (
		totalResources   int
		missingTagsCount int
		noOwnerCount     int
	)

	for {
		response, err := client.SearchResources(ctx, request)
		if err != nil {
			log.Printf("Error searching resources in %s: %v", section, err)
			return
		}

		for _, resource := range response.Items {
			formattedTime, daysSinceCreation := formatTimeCreated(resource.TimeCreated)
			row := []string{
				section,
				getStringValue(resource.DisplayName),
				getStringValue(resource.ResourceType),
				getStringValue(resource.Identifier),
				getStringValue(resource.CompartmentId),
				getStringValue(resource.LifecycleState),
				formattedTime,
				daysSinceCreation,
				getStringValue(resource.AvailabilityDomain),
				DefinedTagsToString(resource.DefinedTags),
				FreeformTagsToString(resource.FreeformTags),
			}

			// Write to main report
			if err := mainWriter.Write(row); err != nil {
				log.Printf("Error writing to main report: %v", err)
				continue
			}

			// Check for missing tags
			if createMissingTagsFile && len(resource.DefinedTags) == 0 {
				if err := missingTagsWriter.Write(row); err != nil {
					log.Printf("Error writing to missing tags report: %v", err)
				} else {
					missingTagsCount++
				}
			}

			// Check for missing owner
			if createNoOwnerFile && (len(resource.DefinedTags) == 0 || !hasCreatedByTag(resource.DefinedTags)) {
				if err := noOwnerWriter.Write(row); err != nil {
					log.Printf("Error writing to no owner report: %v", err)
				} else {
					noOwnerCount++
				}
			}

			totalResources++
		}

		if response.OpcNextPage == nil {
			break
		}
		request.Page = response.OpcNextPage
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("%s: Processed %d resources", section, totalResources)
	if createMissingTagsFile {
		log.Printf("%s: Found %d resources with missing tags", section, missingTagsCount)
	}
	if createNoOwnerFile {
		log.Printf("%s: Found %d resources with no owner", section, noOwnerCount)
	}
}

func main() {
	ctx := context.Background()

	homeKey, err := GetHomeRegionKeyFromDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Error retrieving HomeRegionKey: %v", err)
	}
	log.Printf("HomeRegionKey: %s", homeKey)

	configPath, err := ReadFirstLine("config_path.txt")
	if err != nil {
		log.Fatalf("Error reading config path: %v", err)
	}
	log.Printf("Using config file: %s", configPath)

	cfg, err := ini.Load(configPath)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	var wg sync.WaitGroup
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		wg.Add(1)
		go func(sectionName string) {
			defer wg.Done()
			log.Printf("Processing region: %s", sectionName)
			ExecuteFullSearch(configPath, sectionName, `query all resources`)
		}(section.Name())
	}

	wg.Wait()
	log.Println("All regions processed successfully")
}

//
