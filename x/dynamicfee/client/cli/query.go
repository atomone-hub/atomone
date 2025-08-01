package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/atomone-hub/atomone/x/dynamicfee/types"
)

// GetQueryCmd returns the parent command for all x/dynamicfee cli query commands.
func GetQueryCmd() *cobra.Command {
	// create base command
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// add sub-commands
	cmd.AddCommand(
		GetParamsCmd(),
		GetStateCmd(),
		GetGasPriceCmd(),
		GetGasPricesCmd(),
	)

	return cmd
}

// GetParamsCmd returns the cli-command that queries the current dynamicfee parameters.
func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query for the current dynamicfee parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.Params(cmd.Context(), &types.ParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&resp.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetStateCmd returns the cli-command that queries the current dynamicfee state.
func GetStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Query for the current dynamicfee state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.State(cmd.Context(), &types.StateRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&resp.State)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetGasPriceCmd returns the cli-command that queries the current dynamicfee gas price.
func GetGasPriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-price [denom]",
		Short: "Query for the current dynamicfee gas price",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.GasPrice(cmd.Context(), &types.GasPriceRequest{
				Denom: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetGasPricesCmd returns the cli-command that queries all current dynamicfee gas prices.
func GetGasPricesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-prices",
		Short: "Query for all current dynamicfee gas prices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.GasPrices(cmd.Context(), &types.GasPricesRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
