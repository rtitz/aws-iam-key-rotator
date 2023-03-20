package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/rtitz/aws-iam-key-rotator/awsUtils"
	"github.com/rtitz/aws-iam-key-rotator/variables"
)

func listAccessKeys(ctx context.Context, cfg *aws.Config, awsProfile, awsUser string) (int, error) {
	var numberOfAccessKeys int = 0

	// Create a IAM service client.
	svc_iam := iam.NewFromConfig(*cfg)

	var input *iam.ListAccessKeysInput
	if len(awsUser) != 0 {
		input = &iam.ListAccessKeysInput{UserName: aws.String(awsUser)}
	} else {
		input = &iam.ListAccessKeysInput{}
	}

	result, err := svc_iam.ListAccessKeys(ctx, input)
	if err != nil {
		fmt.Printf("%s : Error: %s\n", awsProfile, err)
		return numberOfAccessKeys, err
	}

	for i, key := range result.AccessKeyMetadata {
		if *key.AccessKeyId == "" {
			continue
		}
		numberOfAccessKeys++
		keyElement[i].accesskey = *key.AccessKeyId
		keyElement[i].username = *key.UserName
		keyElement[i].creationtime = *key.CreateDate
		keyElement[i].status = string(key.Status)
		ageOfKey := time.Since(keyElement[i].creationtime)

		fmt.Printf("%s : IAM user: %s / Current AccessKeyId: %s / Status: %s\n", awsProfile, keyElement[i].username, keyElement[i].accesskey, keyElement[i].status)
		fmt.Printf("%s : Creation time of AccessKey: %s / Age: %s\n", awsProfile, keyElement[i].creationtime, ageOfKey.Truncate(time.Second).String())

		profiles[awsProfile]["userName"] = *key.UserName
		profiles[awsProfile]["oldAccessKeyId"] = *key.AccessKeyId
	}
	return numberOfAccessKeys, err
}

func createAccessKey(ctx context.Context, cfg *aws.Config, awsProfile, awsUser string) (*iam.CreateAccessKeyOutput, error) {
	// Create a IAM service client.
	svc_iam := iam.NewFromConfig(*cfg)
	var input *iam.CreateAccessKeyInput = &iam.CreateAccessKeyInput{UserName: aws.String(awsUser)}

	result, err := svc_iam.CreateAccessKey(ctx, input)
	if err != nil {
		fmt.Printf("%s : Failed to create AccessKey! %s\n", awsProfile, err.Error())
	}
	return result, err
}

func saveNewAccessKey(awsProfile, newAccessKeyId, newSecretAccessKey, awsCmd string) bool {
	var saved bool = false
	if len(awsProfile) == 0 {
		awsProfile = "default"
	}

	if newAccessKeyId == "" {
		return false
	}

	fmt.Printf("%s : New AccessKeyId: %s / New SecretAccessKey: ************\n", awsProfile, newAccessKeyId)

	cmd01 := exec.Command(awsCmd, "--profile", awsProfile, "--output", "json", "configure", "set", "aws_secret_access_key", newSecretAccessKey)
	cmd02 := exec.Command(awsCmd, "--profile", awsProfile, "--output", "json", "configure", "set", "aws_access_key_id", newAccessKeyId)
	cmd := cmd01

	for i := 0; i < 2; i++ {
		if i == 0 {
			cmd = cmd01
		} else if i == 1 {
			cmd = cmd02
		} else {
			break
		}

		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Println(awsProfile + " : " + fmt.Sprint(err) + ": " + stderr.String())
			return saved
		}
	}
	saved = true
	return saved
}

func deleteAccessKey(ctx context.Context, cfg *aws.Config, awsUser, accesskeyid string) error {
	svc_iam := iam.NewFromConfig(*cfg)
	var input *iam.DeleteAccessKeyInput = &iam.DeleteAccessKeyInput{UserName: aws.String(awsUser), AccessKeyId: aws.String(accesskeyid)}
	_, err := svc_iam.DeleteAccessKey(ctx, input)
	return err
}

func listAwsProfiles(awsProfile, awsCmd string) bool {
	if len(awsProfile) == 0 {
		awsProfile = "default"
	}
	cmd := exec.Command(awsCmd, "configure", "list-profiles")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil { // Failed to verify AWS profile with CLI version 2 method, trying again with legacy way
		cmd := exec.Command(awsCmd, "configure", "list", "--profile", awsProfile)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		errLegacy := cmd.Run()
		if errLegacy != nil { // Legacy check also failed / profile not found
			fmt.Printf("\nPLEASE VERIFY THAT THE AWS CLI IS INSTALLED AND CONFIGURED! (SEE: https://docs.aws.amazon.com/cli/)\n\n")
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			return false
		}
		return true
	}

	var profileFound bool = false
	item := bufio.NewScanner(strings.NewReader(out.String()))
	for item.Scan() {
		if strings.HasPrefix(item.Text()+"\n", awsProfile+"\n") {
			profileFound = true
		}
	}
	return profileFound
}

// main
type iamKeysStruct struct {
	accesskey    string
	username     string
	creationtime time.Time
	status       string
}

var keyElement [2]iamKeysStruct

var profiles = map[string]map[string]string{}

var wg sync.WaitGroup

func main() {
	startTime := time.Now()

	// Define and check parameters
	awsProfileArg := flag.String("profile", "", "Specify the AWS CLI profile, for example: 'default' or a comma-separated list like 'default,test'")
	goroutine := flag.Bool("parallel", false, "If specified parallel execution is enabled. By default (not specified) parallel execution is disabled")
	flag.Parse()

	if len(*awsProfileArg) == 0 {
		fmt.Printf("Parameter missing! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		os.Exit(999)
	}
	// End of: Define and check parameters

	fmt.Printf("%s %s\n\n", variables.AppName, variables.AppVersion)
	//fmt.Printf("Runtime: %s (%s)\n", runtime.GOOS, runtime.GOARCH)
	if *goroutine {
		fmt.Printf("Parallel mode: Enabled\n\n")
	} else {
		fmt.Printf("Parallel mode: Disabled\n\n")
	}

	var someProfilesNotFound bool = false
	awsProfiles := strings.Split(*awsProfileArg, ",")

	// Check if awsProfiles are there
	for _, awsProfile := range awsProfiles {
		awsProfile = strings.TrimSpace(awsProfile)
		profiles[awsProfile] = map[string]string{} // Create map per profile
		profileExists := listAwsProfiles(awsProfile, variables.AwsCmd)
		if !profileExists {
			someProfilesNotFound = true
			fmt.Printf("Profile '%s' does not exist!\n", awsProfile)
		}
		if someProfilesNotFound {
			fmt.Printf("List existing profiles with the following command:\n")
			fmt.Printf("%s configure list-profiles\n", variables.AwsCmd)
			os.Exit(900)
		}
	}

	// Start the rotation for each profile
	var returnCode int = 0
	rc := make(chan int, len(awsProfiles))

	for _, awsProfile := range awsProfiles {
		awsProfile = strings.TrimSpace(awsProfile)
		if *goroutine {
			wg.Add(1) // Add one instance to the goroutine waitgroup
			go start(awsProfile, variables.AwsRegion, *goroutine, rc)
			//rcInt += <-rc  // Receiving the channel here will make the goroutine to run in a sequence instead of a parallel
		} else {
			returnCode += start(awsProfile, variables.AwsRegion, *goroutine, rc)
			fmt.Println("") // Empty line as sepatater for nicer output
		}
	}

	if *goroutine {
		wg.Wait() // Wait here until all goroutines are finished
		close(rc) // Close the channel after all goroutines are finished
		for value := range rc {
			//fmt.Println(value) // Debug for return code values
			returnCode += value // Add all values in the channel. If one or more are not 0, returncode will not be 0
		}
	}
	timeForRotation := time.Since(startTime)

	if *goroutine { // Save new key in local configuration, if not done yet because of goroutine
		fmt.Println("\nUpdating local configuration in non-parallel-mode to prevent simultaneous access to same configuration file.")
		for _, awsProfile := range awsProfiles {
			if profiles[awsProfile]["newAccessKeyCreated"] == "YES" {
				saved := saveNewAccessKey(awsProfile, profiles[awsProfile]["newAccessKeyId"], profiles[awsProfile]["newSecretAccessKey"], variables.AwsCmd)
				if saved {
					fmt.Printf("%s : Local AWS CLI configuration updated with new AccessKeyId and SecretAccessKey\n", awsProfile)
				} else {
					fmt.Printf("\nWARNING ! \nCould not save the new AccessKeyId and SecretAccessKey for the following User:\n")
					fmt.Println(awsProfile, "Username", profiles[awsProfile]["userName"])
					fmt.Println(awsProfile, "Old Key", profiles[awsProfile]["oldAccessKeyId"])
					fmt.Println(awsProfile, "New Key", profiles[awsProfile]["newAccessKeyId"])
					fmt.Println(awsProfile, "New Secret", profiles[awsProfile]["newSecretAccessKey"])
				}
			} else {
				fmt.Printf("\nNo new AccessKey stored for %s\n", awsProfile)
			}
		}
		fmt.Println("")
	}
	fmt.Printf("Key rotation for %d AWS profiles done! - Exection time %s seconds\n", len(awsProfiles), timeForRotation)
	fmt.Printf("DONE! - Entire exection time %s seconds\n", time.Since(startTime))
	os.Exit(returnCode)
}

func start(awsProfile, awsRegion string, goroutine bool, rc chan int) int {
	if goroutine {
		defer wg.Done() // Delete one instance from the goroutine waitgroup
	}
	//fmt.Printf("%s : AWS region: %s (global service)\n", awsProfile, awsRegion)

	// Create new session
	ctx := context.TODO()
	cfg, errCreateAwsSession := awsUtils.CreateAwsSession(ctx, awsProfile, awsRegion)
	if errCreateAwsSession != nil {
		fmt.Printf("%s : Failed to create a session! %s\n", awsProfile, errCreateAwsSession.Error())
		//os.Exit(1)
		rc <- 1
		return 1
	}

	// List AccessKeys
	numberOfAccessKeys, errlistAccessKeys := listAccessKeys(ctx, &cfg, awsProfile, "")
	if errlistAccessKeys != nil {
		fmt.Printf("%s : Failed to list access keys! %s\n", awsProfile, errlistAccessKeys.Error())
		rc <- 2
		return 2
	}

	// Create new AccessKey
	profiles[awsProfile]["newAccessKeyCreated"] = "NO"
	if numberOfAccessKeys > 1 {
		fmt.Printf("%s : Can not add any keys. There are %d keys present. 2 is the maximum allowed by AWS!\n", awsProfile, numberOfAccessKeys)
		rc <- 3
		return 3
	} else {
		fmt.Printf("%s : Number of keys: %d / Creating new AccessKey...\n", awsProfile, numberOfAccessKeys)
		result, err := createAccessKey(ctx, &cfg, awsProfile, profiles[awsProfile]["userName"])
		if err != nil {
			fmt.Printf("%s : Failed to create access key! %s\n", awsProfile, err.Error())
			rc <- 4
			return 4
		}
		profiles[awsProfile]["newAccessKeyId"] = *result.AccessKey.AccessKeyId
		profiles[awsProfile]["newSecretAccessKey"] = *result.AccessKey.SecretAccessKey
		profiles[awsProfile]["newAccessKeyCreated"] = "YES"
	}

	// Save new AccessKey in configuration
	var returnCode int = 999 // Unknown result
	if profiles[awsProfile]["newAccessKeyCreated"] == "YES" {
		var saved bool = false
		if goroutine { // if in goroutine, just confirm the configuration save and save later outside of the goroutine
			saved = true
		} else {
			saved = saveNewAccessKey(awsProfile, profiles[awsProfile]["newAccessKeyId"], profiles[awsProfile]["newSecretAccessKey"], variables.AwsCmd)
		}
		// Delete old AccessKey
		var keyToDelete string = profiles[awsProfile]["newAccessKeyId"]
		returnCode = 8 // Key rotation was successful. But not clear if new key was stored successful. Maybe new key will be deleted from AWS

		if saved {
			keyToDelete = profiles[awsProfile]["oldAccessKeyId"]
			returnCode = 0 // Key rotation (incl. storing of new key) was successful. Old key will be deleted from AWS.
			if !goroutine {
				fmt.Printf("%s : Local AWS CLI configuration updated with new AccessKeyId and SecretAccessKey\n", awsProfile)
			}
		} else {
			fmt.Printf("%s : Deleting NEW AccessKey from AWS, because the new one was not saved successfully.\n", awsProfile)
		}
		errdeleteAccessKey := deleteAccessKey(ctx, &cfg, profiles[awsProfile]["userName"], keyToDelete)
		if errdeleteAccessKey == nil {
			fmt.Printf("%s : AccessKey %s successfully deleted\n", awsProfile, keyToDelete)
		} else {
			fmt.Printf("%s : AccessKey deletion failed for key: %s\n", awsProfile, keyToDelete)
			fmt.Printf("%s : AccessKey deletion failed! %s\n", awsProfile, errdeleteAccessKey.Error())
			rc <- 5
			return 5
		}
	} else {
		returnCode = 9 // Key rotation was NOT successful. New key NOT created!
	}

	if goroutine {
		rc <- returnCode // Add returnCode to channel rc
	}
	return returnCode
}
