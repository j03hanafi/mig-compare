package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"math"
	"os"
	"strings"
)

type FileDiff struct {
	FilePathA         string
	FileTypeA         string
	LastModifiedDateA string
	SizeA             int64

	FilePathB         string
	FileTypeB         string
	LastModifiedDateB string
	SizeB             int64
}

func main() {
	branchA := flag.String("source", "", "Name of the first branch to compare")
	branchB := flag.String("target", "", "Name of the second branch to compare")
	repoDir := flag.String("dir", "", "Path to the repository directory")
	output := flag.String("output", "", "Path/name to the output CSV file")

	flag.Parse()

	if *branchA == "" || *branchB == "" {
		fmt.Println("Both branch names must be provided")
		return
	}

	if *repoDir == "" {
		*repoDir = "." // default to current directory
	}

	if *output == "" {
		*output = "comparison" // default to current directory
	}

	*repoDir += "/"
	*output += ".csv"

	fmt.Println("Comparing branches", *branchA, "and", *branchB, "in", *repoDir)

	// Open the existing repository
	r, err := git.PlainOpen(*repoDir)
	if err != nil {
		fmt.Println("Error opening repository:", err)
		return
	}

	// Fetch the last commits of both branches
	commitA, err := getLastCommit(r, *branchA)
	if err != nil {
		fmt.Println("Error fetching last commit for branch", *branchA, ":", err)
		return
	}

	commitB, err := getLastCommit(r, *branchB)
	if err != nil {
		fmt.Println("Error fetching last commit for branch", *branchB, ":", err)
		return
	}

	// Compare files between the two commits
	fileDiffs, err := compareCommits(commitA, commitB, *repoDir)
	if err != nil {
		fmt.Println("Error comparing commits:", err)
		return
	}

	// Extract file information and write to CSV
	err = writeComparisonToCSV(fileDiffs, *branchA, *branchB, *output)
	if err != nil {
		fmt.Println("Error writing to CSV:", err)
		return
	}

	fmt.Println("Successfully wrote comparison to CSV")
}

// getLastCommit retrieves the last commit from a specified branch
func getLastCommit(r *git.Repository, branchName string) (*object.Commit, error) {
	// Find the branch reference
	ref, err := r.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		return nil, fmt.Errorf("could not find branch %s: %w", branchName, err)
	}

	// Get the commit object from the reference
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("could not find commit from reference %s: %w", ref.Hash(), err)
	}

	return commit, nil
}

// compareCommits compares files between two commits and returns a slice of FileDiff
func compareCommits(commitA, commitB *object.Commit, repoDir string) ([]FileDiff, error) {
	// Retrieve the tree for each commit
	treeA, err := commitA.Tree()
	if err != nil {
		return nil, err
	}

	treeB, err := commitB.Tree()
	if err != nil {
		return nil, err
	}

	// Compare the trees
	changes, err := object.DiffTreeWithOptions(context.Background(), treeA, treeB, object.DefaultDiffTreeOptions)
	if err != nil {
		return nil, err
	}

	// Process each change to create a slice of FileDiff
	var diffs []FileDiff
	failCount := 0
	for _, change := range changes {
		fileDiff, err := processChange(change, repoDir)
		if err != nil {
			fmt.Println("Error processing change:", err)
			failCount++
			continue
		}
		diffs = append(diffs, fileDiff)
	}

	if failCount > 0 {
		fmt.Printf("Failed to process %d changes\n", failCount)
		return diffs, errors.New(fmt.Sprintf("failed to process %d changes", failCount))
	}

	return diffs, nil
}

// processChange processes a change object and returns a FileDiff
func processChange(change *object.Change, repoDir string) (FileDiff, error) {
	var diff FileDiff
	var err error

	// Get file information from the 'From' side of the change (if it exists)
	if change.From.Name != "" {
		diff.FilePathA = change.From.Name
		fileTypeA := strings.Split(change.From.Name, ".")
		diff.FileTypeA = strings.ToUpper(fileTypeA[len(fileTypeA)-1])
		diff.LastModifiedDateA, diff.SizeA, err = getFileDetails(repoDir + change.From.Name)
		if err != nil {
			return FileDiff{}, err
		}
	}

	// Get file information from the 'To' side of the change (if it exists)
	if change.To.Name != "" {
		diff.FilePathB = change.To.Name
		fileTypeB := strings.Split(change.To.Name, ".")
		diff.FileTypeB = strings.ToUpper(fileTypeB[len(fileTypeB)-1])
		diff.LastModifiedDateB, diff.SizeB, err = getFileDetails(repoDir + change.To.Name)
		if err != nil {
			return FileDiff{}, err
		}
	}

	return diff, nil
}

// getFileDetails fetches the last modification date and size of a file
func getFileDetails(filePath string) (lastModifiedDate string, sizeKB int64, err error) {
	file, err := os.Stat(filePath)
	if err != nil {
		return "", 0, err // Return an error if the file cannot be accessed
	}

	// Get the last modified date in a readable format
	lastModifiedDate = file.ModTime().Format("02/01/2006")

	// Get the file size in kilobytes and round up
	sizeBytes := file.Size()
	sizeKB = int64(math.Ceil(float64(sizeBytes) / 1024.0))

	return lastModifiedDate, sizeKB, nil
}

// writeComparisonToCSV writes the comparison results to a CSV file
func writeComparisonToCSV(fileDiffs []FileDiff, branchAName string, branchBName string, outputFileName string) error {
	// Create a new CSV file
	file, err := os.Create(outputFileName)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the branch names as headers
	branchHeaders := []string{branchAName, "", "", "", branchBName, "", "", ""}
	if err := writer.Write(branchHeaders); err != nil {
		return err
	}

	// Write the sub-headers for each branch
	subHeaders := []string{
		"Library/Object", "Type", "Compile/Promote Date", "Size (KBytes)",
		"Library/Object", "Type", "Compile/Promote Date", "Size (KBytes)",
	}
	if err := writer.Write(subHeaders); err != nil {
		return err
	}

	// Write each file diff to the CSV
	for _, diff := range fileDiffs {
		row := []string{
			diff.FilePathA, diff.FileTypeA, diff.LastModifiedDateA, fmt.Sprintf("%d", diff.SizeA),
			diff.FilePathB, diff.FileTypeB, diff.LastModifiedDateB, fmt.Sprintf("%d", diff.SizeB),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
