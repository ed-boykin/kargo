package warehouses

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
)

func TestSelectCharts(t *testing.T) {
	testCases := []struct {
		name                 string
		credentialsDB        credentials.Database
		selectChartVersionFn func(
			context.Context,
			string,
			string,
			string,
			*helm.Credentials,
		) (string, error)
		assertions func(*testing.T, []kargoapi.Chart, error)
	}{
		{
			name: "error getting repository credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false,
						errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Chart, err error) {
				require.ErrorContains(t, err, "error obtaining credentials for chart")
				require.ErrorContains(t, err, "something went wrong")
			},
		},

		{
			name: "error getting latest chart version",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			selectChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(t *testing.T, _ []kargoapi.Chart, err error) {
				require.ErrorContains(t, err, "error searching for latest version of chart")
				require.ErrorContains(t, err, "something went wrong")
			},
		},

		{
			name: "no chart found",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			selectChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "", nil
			},
			assertions: func(t *testing.T, _ []kargoapi.Chart, err error) {
				require.ErrorContains(t, err, "found no suitable version of chart")
			},
		},

		{
			name: "success",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			selectChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "1.0.0", nil
			},
			assertions: func(t *testing.T, charts []kargoapi.Chart, err error) {
				require.NoError(t, err)
				require.Len(t, charts, 1)
				require.Equal(
					t,
					kargoapi.Chart{
						RepoURL: "fake-url",
						Name:    "fake-chart",
						Version: "1.0.0",
					},
					charts[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			charts, err := (&reconciler{
				credentialsDB:        testCase.credentialsDB,
				selectChartVersionFn: testCase.selectChartVersionFn,
			}).selectCharts(
				context.Background(),
				"fake-namespace",
				[]kargoapi.RepoSubscription{
					{
						Chart: &kargoapi.ChartSubscription{
							RepoURL: "fake-url",
							Name:    "fake-chart",
						},
					},
				},
			)
			testCase.assertions(t, charts, err)
		})
	}
}
