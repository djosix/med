/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/golang/snappy"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use: "test",
	Run: mainForTest,
}

func init() {
	clientCmd.AddCommand(testCmd)
}

func mainForTest(cmd *cobra.Command, args []string) {
	_, _ = cmd, args
	// message := pb.Message{
	// 	Content: &pb.Message_C1{
	// 		C1: &pb.Content1{
	// 			// Name: "test",
	// 		},
	// 	},
	// }
	// data, err := proto.Marshal(&message)
	// if err != nil {
	// 	fmt.Println("error:", err)
	// 	return
	// }
	// fmt.Println("Message:", data)
	data := []byte{}
	for len(data) < 1000 {
		data = append(data, byte(len(data)%256))
	}

	encoded := []byte{}
	encoded = snappy.Encode(encoded, data)

	fmt.Println("src:", len(data))
	fmt.Println("dst:", len(encoded))
}
