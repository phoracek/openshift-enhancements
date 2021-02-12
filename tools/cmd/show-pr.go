package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/openshift/enhancements/tools/enhancements"
	"github.com/openshift/enhancements/tools/report"
	"github.com/openshift/enhancements/tools/stats"
	"github.com/openshift/enhancements/tools/util"
)

func newShowPRCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "show-pr",
		Short:     "Dump details for a pull request",
		ValidArgs: []string{"pull-request-id"},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("please specify one valid pull request ID")
			}
			if _, err := strconv.Atoi(args[0]); err != nil {
				return errors.Wrap(err,
					fmt.Sprintf("pull request ID %q must be an integer", args[0]))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return errors.Wrap(err,
					fmt.Sprintf("failed to interpret pull request ID %q as a number", args[0]))
			}
			group, isEnhancement, err := enhancements.GetGroup(prID)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to determine group for PR %d", prID))
			}

			fmt.Printf("Group: %s\n", group)
			fmt.Printf("Enhancement: %v\n", isEnhancement)

			ghClient := util.NewGithubClient(configSettings.Github.Token)
			ctx := context.Background()
			pr, _, err := ghClient.PullRequests.Get(ctx, orgName, repoName, prID)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to fetch pull request %d", prID))
			}

			query := &util.PullRequestQuery{
				Org:     orgName,
				Repo:    repoName,
				DevMode: false,
				Client:  ghClient,
			}

			// Set up a Stats object so we can get the details for the
			// pull request.
			//
			// TODO: This is a bit clunky. Can we improve it without
			// forcing the low level report code to know all about
			// everything?
			all := stats.Bucket{
				Rule: func(prd *stats.PullRequestDetails) bool {
					return true
				},
			}
			reportBuckets := []*stats.Bucket{
				&all,
			}
			theStats := &stats.Stats{
				Query:   query,
				Buckets: reportBuckets,
			}
			if err := theStats.ProcessOne(pr); err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to fetch details for PR %d", prID))
			}

			report.ShowPRs(
				fmt.Sprintf("Pull Request %d", prID),
				all.Requests,
				true,
			)

			prd := all.Requests[0]
			fmt.Printf("State:       %q\n", prd.State)
			fmt.Printf("LGTM:        %v\n", prd.LGTM)
			fmt.Printf("Prioritized: %v\n", prd.Prioritized)
			fmt.Printf("Stale:       %v\n", prd.Stale)

			return nil
		},
	}

	return cmd
}

func init() {
	rootCmd.AddCommand(newShowPRCommand())
}