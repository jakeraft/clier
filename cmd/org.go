package cmd

import (
	"strconv"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newOrgCmd())
}

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "org",
		Short:   "Manage organizations",
		GroupID: rootGroupSettings,
		Long:    `Create, delete, and manage organization membership.`,
	}
	cmd.AddCommand(newOrgCreateCmd())
	cmd.AddCommand(newOrgDeleteCmd())
	cmd.AddCommand(newOrgListCmd())
	cmd.AddCommand(newOrgMembersCmd())
	cmd.AddCommand(newOrgInviteCmd())
	cmd.AddCommand(newOrgRemoveCmd())
	return cmd
}

func newOrgCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			resp, err := client.CreateOrg(api.CreateOrgRequest{Name: args[0]})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}

func newOrgDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			if err := client.DeleteOrg(args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your organizations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			orgs, err := client.ListMyOrgs()
			if err != nil {
				return err
			}
			return printJSON(orgs)
		},
	}
}

func newOrgMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members <org-name>",
		Short: "List organization members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			members, err := client.ListOrgMembers(args[0])
			if err != nil {
				return err
			}
			return printJSON(members)
		},
	}
}

func newOrgInviteCmd() *cobra.Command {
	var role int

	cmd := &cobra.Command{
		Use:   "invite <org-name> <user-name>",
		Short: "Invite a user to an organization",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			if err := client.InviteOrgMember(args[0], api.InviteMemberRequest{
				Name: args[1],
				Role: role,
			}); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "invited",
				"org":    args[0],
				"user":   args[1],
				"role":   strconv.Itoa(role),
			})
		},
	}
	cmd.Flags().IntVar(&role, "role", 0, "Member role (0=member, 1=admin)")
	return cmd
}

func newOrgRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <org-name> <user-name>",
		Short: "Remove a user from an organization",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			if err := client.RemoveOrgMember(args[0], args[1]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status":  "removed",
				"org":     args[0],
				"removed": args[1],
			})
		},
	}
}
