package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

type DataPoint struct {
	DataType string `json:"data_type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
}

type DataPointRequest struct {
	DatasetID  string      `json:"dataset_id"`
	DataPoints []DataPoint `json:"datapoints"`
}

type Dataset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DatasetResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

var (
	datasetID   string
	searchieURL string
)

var rootCmd = &cobra.Command{
	Use:   "searchie-fs-importer [folder]",
	Short: "Import files into Searchie",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		folder := args[0]
		client := resty.New()

		if datasetID == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("No dataset ID provided. Would you like to create a new dataset? (y/n): ")
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				fmt.Print("Enter dataset name: ")
				name, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("error reading dataset name: %w", err)
				}
				name = strings.TrimSpace(name)

				var datasetResp DatasetResponse
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(map[string]string{"name": name}).
					SetResult(&datasetResp).
					Post(searchieURL + "/api/datasets")

				if err != nil {
					return fmt.Errorf("error creating dataset: %w", err)
				}

				if resp.IsError() {
					return fmt.Errorf("error creating dataset: %s", resp.String())
				}

				datasetID = datasetResp.Data.ID
				fmt.Printf("Created dataset with ID: %s\n", datasetID)
			} else {
				return fmt.Errorf("dataset ID is required")
			}
		}

		files := []string{}
		err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error walking directory: %w", err)
		}

		bar := progressbar.Default(int64(len(files)))

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("error reading file %s: %w", file, err)
			}

			request := DataPointRequest{
				DatasetID: datasetID,
				DataPoints: []DataPoint{
					{
						DataType: "text",
						Name:     strings.TrimSuffix(strings.TrimPrefix(file, folder+"/"), filepath.Ext(file)),
						Data:     string(content),
					},
				},
			}

			resp, err := client.R().
				SetHeader("Content-Type", "application/json").
				SetBody(request).
				Post(searchieURL + "/api/datapoints")

			if err != nil {
				return fmt.Errorf("error making request for file %s: %w", file, err)
			}

			if resp.IsError() {
				fmt.Printf("error response for file %s: %s\n", file, resp.String())
				continue
			}

			bar.Add(1)
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&datasetID, "dataset-id", "", "Dataset ID to import into")
	rootCmd.Flags().StringVar(&searchieURL, "searchie-url", "", "Searchie API URL")
	rootCmd.MarkFlagRequired("searchie-url")
}
