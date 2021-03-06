/*
Copyright (c) 2020 Bitnami

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Currently these tests will be skipped entirely unless the
// ENABLE_PG_INTEGRATION_TESTS env var is set.
// Run the local postgres with
// docker run --publish 5432:5432 -e ALLOW_EMPTY_PASSWORD=yes bitnami/postgresql:11.6.0-debian-9-r0
// in another terminal.
package main

import (
	"database/sql"
	"testing"

	"github.com/kubeapps/kubeapps/pkg/chart/models"
	"github.com/kubeapps/kubeapps/pkg/dbutils/dbutilstest/pgtest"
	_ "github.com/lib/pq"
)

func getInitializedManager(t *testing.T) (*postgresAssetManager, func()) {
	pam, cleanup := pgtest.GetInitializedManager(t)
	return &postgresAssetManager{pam}, cleanup
}

func TestGetChart(t *testing.T) {
	pgtest.SkipIfNoDB(t)
	const repoName = "repo-name"

	testCases := []struct {
		name string
		// existingCharts is a map of charts per namespace
		existingCharts map[string][]models.Chart
		chartId        string
		namespace      string
		expectedChart  string
		expectedErr    error
	}{
		{
			name:        "it returns an error if the chart does not exist",
			chartId:     "doesnt-exist-1",
			namespace:   "doesnt-exist",
			expectedErr: sql.ErrNoRows,
		},
		{
			name: "it returns the chart matching the chartid",
			existingCharts: map[string][]models.Chart{
				"namespace-1": []models.Chart{
					models.Chart{ID: "chart-1", Name: "my-chart"},
				},
			},
			chartId:       "chart-1",
			namespace:     "namespace-1",
			expectedErr:   nil,
			expectedChart: "my-chart",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pam, cleanup := getInitializedManager(t)
			defer cleanup()
			for namespace, charts := range tc.existingCharts {
				pgtest.EnsureChartsExist(t, pam, charts, models.Repo{Name: repoName, Namespace: namespace})
			}

			chart, err := pam.getChart(tc.chartId)

			if got, want := err, tc.expectedErr; got != want {
				t.Fatalf("got: %+v, want: %+v", got, want)
			}
			if got, want := chart.Name, tc.expectedChart; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	pgtest.SkipIfNoDB(t)
	const repoName = "repo-name"

	testCases := []struct {
		name string
		// existingCharts is a map of charts per namespace
		existingCharts   map[string][]models.Chart
		chartId          string
		namespace        string
		requestedVersion string
		expectedVersion  string
		expectedErr      error
	}{
		{
			name:        "it returns an error if the chart does not exist",
			chartId:     "doesnt-exist-1",
			namespace:   "doesnt-exist",
			expectedErr: sql.ErrNoRows,
		},
		{
			name: "it returns an error if the chart version does not exist",
			existingCharts: map[string][]models.Chart{
				"namespace-1": []models.Chart{
					models.Chart{ID: "chart-1", ChartVersions: []models.ChartVersion{
						models.ChartVersion{Version: "1.2.3"},
					}},
				},
			},
			chartId:          "chart-1",
			namespace:        "namespace-1",
			requestedVersion: "doesnt-exist",
			expectedErr:      ErrChartVersionNotFound,
		},
		{
			name: "it returns the chart version matching the chartid and version",
			existingCharts: map[string][]models.Chart{
				"namespace-1": []models.Chart{
					models.Chart{ID: "chart-1", ChartVersions: []models.ChartVersion{
						models.ChartVersion{Version: "1.2.3"},
						models.ChartVersion{Version: "4.5.6"},
					}},
				},
			},
			chartId:          "chart-1",
			namespace:        "namespace-1",
			requestedVersion: "1.2.3",
			expectedVersion:  "1.2.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pam, cleanup := getInitializedManager(t)
			defer cleanup()
			for namespace, charts := range tc.existingCharts {
				pgtest.EnsureChartsExist(t, pam, charts, models.Repo{Name: repoName, Namespace: namespace})
			}

			chart, err := pam.getChartVersion(tc.chartId, tc.requestedVersion)

			if got, want := err, tc.expectedErr; got != want {
				t.Fatalf("got: %+v, want: %+v", got, want)
			}
			if tc.expectedErr != nil {
				return
			}
			// The function just returns the chart with only the one version.
			if got, want := len(chart.ChartVersions), 1; got != want {
				t.Fatalf("got: %d, want: %d", got, want)
			}
			if got, want := chart.ChartVersions[0].Version, tc.expectedVersion; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
