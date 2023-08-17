package add

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Add issues to sprint.`
	examples = `$ jira sprint add SPRINT_ID ISSUE-1 ISSUE-2`
)

// NewCmdAdd is an add command.
func NewCmdAdd() *cobra.Command {
	return &cobra.Command{
		Use:     "add SPRINT_ID ISSUE-1 [...ISSUE-N]",
		Short:   "Add issues to sprint",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"assign"},
		Annotations: map[string]string{
			"help:args": "SPRINT_ID\t\tID of the sprint on which you want to assign issues to, eg: 123\n" +
				"ISSUE-1 [...ISSUE-N]\tKey of the issues to add to the sprint (max 50 issues at once)",
		},
		Run: add,
	}
}

func add(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")
	boardID := viper.GetInt("board.id")
	params := parseFlags(cmd.Flags(), args, project, boardID)
	client := api.DefaultClient(params.debug)

	qs := getQuestions(params)
	if len(qs) > 0 {
		ans := struct {
			SprintID string
			Issues   string
		}{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.sprintID == "" {
			params.sprintID = ans.SprintID
		}

		if len(params.issues) == 0 {
			issues := strings.Split(ans.Issues, ",")
			for i, iss := range issues {
				issues[i] = cmdutil.GetJiraIssueKey(project, strings.TrimSpace(iss))
			}
			params.issues = issues
		}
	}

	err := func() error {
		s := cmdutil.Info("Adding issues to the sprint...")
		defer s.Stop()

		return client.SprintIssuesAdd(params.sprintID, params.issues...)
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success(fmt.Sprintf("Issues added to the sprint %s\n%s", params.sprintID, cmdutil.GenerateServerBrowseURL(server, project)))
}

func parseFlags(flags query.FlagParser, args []string, project string, boardID int) *addParams {
	var (
		sprintID string
		issues   []string
		tickets  []string
	)

	next, err := flags.GetBool("next")
	cmdutil.ExitIfError(err)

	prev, err := flags.GetBool("prev")
	cmdutil.ExitIfError(err)

	current, err := flags.GetBool("current")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	sprintQuery, err := query.NewSprint(flags)
	cmdutil.ExitIfError(err)
	nArgs := len(args)

	if next || prev || current {
		sprints := func() []*jira.Sprint {

			s := cmdutil.Info("Fetching sprints...")
			defer s.Stop()
			client := api.DefaultClient(debug)

			return client.SprintsInBoards([]int{boardID}, sprintQuery.Get(), 50)
		}()
		sprint := sprints[0]
		if next {
			sprint = sprints[len(sprints)-1]
		}
		sprintID = strconv.Itoa(sprint.ID)
		if nArgs > 0 {
			tickets = args
		}
	} else {
		if nArgs > 0 {
			sprintID = args[0]
		}
		if nArgs > 1 {
			tickets = args[1:]
		}
	}
	issues = make([]string, 0, len(tickets))
	for _, iss := range tickets {
		issues = append(issues, cmdutil.GetJiraIssueKey(project, iss))
	}

	return &addParams{
		sprintID: sprintID,
		issues:   issues,
		debug:    debug,
	}
}

func getQuestions(params *addParams) []*survey.Question {
	var qs []*survey.Question

	if params.sprintID == "" {
		qs = append(qs, &survey.Question{
			Name:     "sprintID",
			Prompt:   &survey.Input{Message: "Sprint ID"},
			Validate: survey.Required,
		})
	}
	if len(params.issues) == 0 {
		qs = append(qs, &survey.Question{
			Name: "issues",
			Prompt: &survey.Input{
				Message: "Issues",
				Help:    "Comma separated list of issues key to add. eg: ISSUE-1, ISSUE-2",
			},
			Validate: survey.Required,
		})
	}

	return qs
}
func setFlags(cmd *cobra.Command) {
	cmd.Flags().String("state", "", "Filter sprint by its state (comma separated).\n"+
		"Valid values are future, active and closed.\n"+
		`Defaults to "active,closed"`)
	cmd.Flags().Bool("show-all-issues", false, "Show sprint issues from all projects")
	cmd.Flags().Bool("table", false, "Display sprints in a table view")
	cmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
		fmt.Sprintf("Accepts (for sprint list): %s", strings.Join(view.ValidSprintColumns(), ", "))+
		fmt.Sprintf("\nAccepts (for sprint issues): %s", strings.Join(view.ValidIssueColumns(), ", ")))
	cmd.Flags().Uint("fixed-columns", 1, "Number of fixed columns in the interactive mode")
	cmd.Flags().Bool("current", false, "List issues in current active sprint")
	cmd.Flags().Bool("prev", false, "List issues in previous sprint")
	cmd.Flags().Bool("next", false, "List issues in next planned sprint")
}
func SetFlags(cmd *cobra.Command) {
	setFlags(cmd)
	// hideFlags(cmd)
}

type addParams struct {
	sprintID string
	issues   []string
	debug    bool
}
