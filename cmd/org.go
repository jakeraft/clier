package cmd

import (
	"strconv"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
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
		RunE:    subcommandRequired,
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
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}

			resp, err := svc.CreateOrg(remoteapi.CreateOrgRequest{Name: args[0]})
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.OrgOf(resp))
		},
	}
}

func newOrgDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}

			if err := svc.DeleteOrg(args[0]); err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.DeletedOf(args[0]))
		},
	}
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your organizations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}

			orgs, err := svc.ListMyOrgs()
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ItemsOf(orgs))
		},
	}
}

func newOrgMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members <org-name>",
		Short: "List organization members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}

			members, err := svc.ListOrgMembers(args[0])
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ItemsOf(members))
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
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}

			if err := svc.InviteOrgMember(args[0], remoteapi.InviteMemberRequest{
				Name: args[1],
				Role: role,
			}); err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.OrgInviteOf(args[0], args[1], strconv.Itoa(role)))
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
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}

			if err := svc.RemoveOrgMember(args[0], args[1]); err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.OrgRemoveOf(args[0], args[1]))
		},
	}
}
