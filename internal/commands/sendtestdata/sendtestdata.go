/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package sendtestdata

import (
	"encoding/json"
	"fmt"

	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const TestDataPath = "/observe-agent/test"

var defaultTestData = map[string]any{
	"hello": "world",
}

func NewSendTestDataCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send-test-data",
		Short: "Sends test data to Observe",
		Long:  "Sends test data to Observe",
		RunE: func(cmd *cobra.Command, args []string) error {
			var testData map[string]any
			dataFlag, _ := cmd.Flags().GetString("data")
			if dataFlag != "" {
				err := json.Unmarshal([]byte(dataFlag), &testData)
				if err != nil {
					return err
				}
			} else {
				testData = defaultTestData
			}
			respBody, err := PostTestDataToObserve(testData, TestDataPath, viper.GetViper())
			if err != nil {
				return err
			}
			fmt.Printf("Successfully sent test data. Saw response: %s\n", respBody)
			return nil
		},
	}
}

func init() {
	sendTestDataCmd := NewSendTestDataCmd()
	RegisterTestDataFlags(sendTestDataCmd)
	root.RootCmd.AddCommand(sendTestDataCmd)
}

func RegisterTestDataFlags(cmd *cobra.Command) {
	cmd.Flags().String("data", "", "specify a given json object to send")
}
